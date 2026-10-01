package evaluation

import (
	"context"
	"time"

	"github.com/stocker-app/stocker/internal/domain"
	"github.com/stocker-app/stocker/internal/store"
)

type Service struct{ db *store.Mongo }

func NewService(db *store.Mongo) *Service { return &Service{db: db} }

func (service *Service) Report(ctx context.Context, asOf time.Time, limit int) (domain.EvaluationReport, error) {
	cases, err := service.db.EvaluationCases(ctx, asOf.UTC(), limit)
	if err != nil {
		return domain.EvaluationReport{}, err
	}
	report := Evaluate(cases, asOf.UTC())
	if err := service.db.SaveEvaluationReport(ctx, report); err != nil {
		return domain.EvaluationReport{}, err
	}
	return report, nil
}
