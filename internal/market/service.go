package market

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/stocker-app/stocker/internal/config"
	"github.com/stocker-app/stocker/internal/domain"
	"github.com/stocker-app/stocker/internal/store"
)

type Service struct {
	store    *store.Mongo
	provider Provider
	maxAge   time.Duration
	log      *slog.Logger
}

func NewService(cfg config.Config, database *store.Mongo, log *slog.Logger) (*Service, error) {
	var provider Provider = FixtureProvider{}
	if cfg.MarketProvider == "indian-stock-api" {
		httpProvider, err := NewPublicHTTPProvider(cfg.MarketEndpoint, cfg.MarketRequestTimeout)
		if err != nil {
			return nil, err
		}
		provider = httpProvider
	}
	return &Service{store: database, provider: provider, maxAge: cfg.MarketRefreshInterval, log: log}, nil
}

func (service *Service) ProviderName() string { return service.provider.Name() }

func (service *Service) Search(ctx context.Context, query string) ([]domain.Security, error) {
	local, err := service.store.SearchSecurities(ctx, query, 20)
	if err != nil {
		return nil, err
	}
	remote, providerErr := service.provider.Search(ctx, query)
	if providerErr != nil {
		service.log.Warn("market_search_provider_failed", "provider", service.provider.Name(), "error", providerErr)
		return local, nil
	}
	for _, instrument := range remote {
		if err := service.store.UpsertMarketSecurity(ctx, instrument.NSESymbol, instrument.BSECode, instrument.ISIN, instrument.CompanyName, instrument.Sector, instrument.Industry, instrument.Source, instrument.SourceURL); err != nil {
			return nil, err
		}
	}
	if len(remote) == 0 {
		return local, nil
	}
	return service.store.SearchSecurities(ctx, query, 20)
}

func (service *Service) Refresh(ctx context.Context, symbol string, force bool) error {
	clean, ok := CleanSymbol(symbol)
	if !ok {
		return store.ErrNotFound
	}
	if !force {
		fresh, err := service.store.MarketQuoteFresh(ctx, clean, service.maxAge)
		if err != nil {
			return err
		}
		if fresh {
			return nil
		}
	}
	snapshot, err := service.provider.Snapshot(ctx, clean)
	if errors.Is(err, ErrNotFound) {
		return store.ErrNotFound
	}
	if err != nil {
		return err
	}
	instrument := snapshot.Instrument
	if instrument.NSESymbol == "" {
		instrument.NSESymbol = clean
	}
	if err := service.store.UpsertMarketSecurity(ctx, instrument.NSESymbol, instrument.BSECode, instrument.ISIN, instrument.CompanyName, instrument.Sector, instrument.Industry, instrument.Source, instrument.SourceURL); err != nil {
		return err
	}
	return service.store.SaveMarketSnapshot(ctx, instrument.NSESymbol, snapshot.Quote, snapshot.Fundamentals)
}

func (service *Service) RefreshBestEffort(ctx context.Context, symbol string) {
	if err := service.Refresh(ctx, symbol, false); err != nil && !errors.Is(err, store.ErrNotFound) {
		service.log.Warn("market_refresh_failed", "provider", service.provider.Name(), "symbol", strings.ToUpper(strings.TrimSpace(symbol)), "error", err)
	}
}
