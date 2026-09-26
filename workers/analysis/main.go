package main

// The analysis worker is deployed independently so model calls can be scaled,
// budgeted, and isolated from ingestion. Phase 3 wires a configured AI.Provider
// to Redis Streams consumer groups and persists schema-validated outputs.
func main() {}
