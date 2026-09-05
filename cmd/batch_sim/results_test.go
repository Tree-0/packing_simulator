package main

import (
	"math"
	"strconv"
	"testing"

	"packing_simulator/backend/evaluator"
	"packing_simulator/backend/policy"
)

func TestRunExperimentProducesResultsForEveryCombination(t *testing.T) {
	config := experimentConfig{
		Workloads: []workloadConfig{
			{Name: "small", Simulation: testSimulationConfig(1, 1)},
			{Name: "wide", Simulation: testSimulationConfig(2, 1)},
		},
		Seeds:      []int64{41, 42},
		Policies:   []string{policy.BottomLeftPolicyName, policy.LargestAreaBottomLeftPolicyName},
		Evaluators: []string{"utilization", "fragmentation"},
		Workers:    1,
	}

	results, err := runExperiment(config)
	if err != nil {
		t.Fatal(err)
	}

	wantRuns := len(config.Workloads) * len(config.Seeds) * len(config.Policies)
	if len(results) != wantRuns {
		t.Fatalf("runExperiment() produced %d runs; want %d", len(results), wantRuns)
	}

	seen := make(map[string]bool, wantRuns)
	for _, result := range results {
		key := result.WorkloadName + "/" + result.PolicyName + "/" + strconv.FormatInt(result.Seed, 10)
		if seen[key] {
			t.Errorf("duplicate result for %s", key)
		}
		seen[key] = true

		if len(result.Evaluations) != len(config.Evaluators) {
			t.Errorf("result %s has %d evaluations; want %d", key, len(result.Evaluations), len(config.Evaluators))
		}

		gotEvaluationTypes := make(map[evaluator.EvaluationType]bool, len(result.Evaluations))
		for _, evaluation := range result.Evaluations {
			gotEvaluationTypes[evaluation.evaluation] = true
		}
		if !gotEvaluationTypes[evaluator.ContainerUtilization] || !gotEvaluationTypes[evaluator.ContainerFragmentation] {
			t.Errorf("result %s evaluations = %v; want utilization and fragmentation", key, result.Evaluations)
		}
	}
}

func TestAggregateResultsCalculatesMeansPerWorkloadPolicyAndEvaluation(t *testing.T) {
	runResults := []RunResult{
		{WorkloadName: "small", PolicyName: policy.BottomLeftPolicyName, Seed: 1, Evaluations: []evaluationResult{
			{evaluation: evaluator.ContainerUtilization, value: 0.25},
			{evaluation: evaluator.ContainerFragmentation, value: 0.60},
		}},
		{WorkloadName: "small", PolicyName: policy.BottomLeftPolicyName, Seed: 2, Evaluations: []evaluationResult{
			{evaluation: evaluator.ContainerUtilization, value: 0.75},
			{evaluation: evaluator.ContainerFragmentation, value: 0.20},
		}},
		{WorkloadName: "large", PolicyName: policy.BottomLeftPolicyName, Seed: 1, Evaluations: []evaluationResult{
			{evaluation: evaluator.ContainerUtilization, value: 0.50},
			{evaluation: evaluator.ContainerFragmentation, value: 0.10},
		}},
		{WorkloadName: "large", PolicyName: policy.BottomLeftPolicyName, Seed: 2, Evaluations: []evaluationResult{
			{evaluation: evaluator.ContainerUtilization, value: 0.90},
			{evaluation: evaluator.ContainerFragmentation, value: 0.30},
		}},
	}

	aggregates, err := AggregateResults(runResults)
	if err != nil {
		t.Fatal(err)
	}

	want := map[AggregateKey]float64{
		{WorkloadName: "small", PolicyName: policy.BottomLeftPolicyName, EvaluationType: evaluator.ContainerUtilization}:   0.50,
		{WorkloadName: "small", PolicyName: policy.BottomLeftPolicyName, EvaluationType: evaluator.ContainerFragmentation}: 0.40,
		{WorkloadName: "large", PolicyName: policy.BottomLeftPolicyName, EvaluationType: evaluator.ContainerUtilization}:   0.70,
		{WorkloadName: "large", PolicyName: policy.BottomLeftPolicyName, EvaluationType: evaluator.ContainerFragmentation}: 0.20,
	}
	if len(aggregates) != len(want) {
		t.Fatalf("AggregateResults() produced %d aggregates; want %d", len(aggregates), len(want))
	}

	for _, aggregate := range aggregates {
		key := AggregateKey{
			WorkloadName:   aggregate.WorkloadName,
			PolicyName:     aggregate.PolicyName,
			EvaluationType: aggregate.Evaluation.evaluation,
		}
		wantValue, found := want[key]
		if !found {
			t.Errorf("unexpected aggregate for %+v", key)
			continue
		}
		if math.Abs(aggregate.Evaluation.value-wantValue) > 1e-9 {
			t.Errorf("aggregate for %+v = %v; want %v", key, aggregate.Evaluation.value, wantValue)
		}
		delete(want, key)
	}

	for missing := range want {
		t.Errorf("missing aggregate for %+v", missing)
	}
}

func testSimulationConfig(maxWidth, maxHeight int) batchSimulationConfig {
	return batchSimulationConfig{
		ContainerHeight:  4,
		ContainerWidth:   4,
		QueueSize:        1,
		MinBoxHeight:     1,
		MaxBoxHeight:     maxHeight,
		MinBoxWidth:      1,
		MaxBoxWidth:      maxWidth,
		Iterations:       3,
		AllowBoxRotation: true,
	}
}
