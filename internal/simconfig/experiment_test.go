package simconfig

import (
	"testing"

	"packing_simulator/backend/evaluator"
	"packing_simulator/backend/policy"
)

func TestExperimentRunSpecsPreservePermutationOrder(t *testing.T) {
	config := ExperimentFile{
		Workloads: []ExperimentWorkload{
			{Name: "one", Simulation: testWorkloadSimulation()},
			{Name: "two", Simulation: testWorkloadSimulation()},
		},
		MaxContainers: 2,
		Seeds:         []int64{7, 8},
		Policies:      []string{policy.BottomLeftPolicyName},
		ContainerSelectors: []ContainerSelectorConfig{
			{Name: policy.ContainerSelectorFirstFitName},
			{Name: policy.ContainerSelectorNextKFitName, K: 2},
		},
		Evaluators: []string{"utilization", "bin-count"},
		Workers:    1,
	}

	specs, err := config.RunSpecs()
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 8 {
		t.Fatalf("RunSpecs() returned %d specs; want 8", len(specs))
	}
	if specs[0].WorkloadName != "one" || specs[0].Config.Seed != 7 || specs[0].ContainerSelector.Name != policy.ContainerSelectorFirstFitName {
		t.Errorf("first spec = %+v", specs[0])
	}
	if specs[3].Config.Seed != 8 || specs[3].ContainerSelector.Name != policy.ContainerSelectorNextKFitName {
		t.Errorf("fourth spec = %+v", specs[3])
	}
	if specs[4].WorkloadName != "two" || specs[4].Index != 4 {
		t.Errorf("fifth spec = %+v", specs[4])
	}
	if len(specs[0].Evaluations) != 2 || specs[0].Evaluations[0] != evaluator.ContainerUtilization || specs[0].Evaluations[1] != evaluator.BinCount {
		t.Errorf("evaluations = %v", specs[0].Evaluations)
	}
}

func TestExperimentRunSpecsDefaultToFirstFit(t *testing.T) {
	config := ExperimentFile{
		Workloads:  []ExperimentWorkload{{Name: "one", Simulation: testWorkloadSimulation()}},
		Seeds:      []int64{1},
		Policies:   []string{policy.BottomLeftPolicyName},
		Evaluators: []string{"utilization"},
	}
	specs, err := config.RunSpecs()
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 1 || specs[0].ContainerSelector.Name != policy.ContainerSelectorFirstFitName {
		t.Errorf("RunSpecs() = %+v; want one First Fit spec", specs)
	}
}

func testWorkloadSimulation() WorkloadSimulation {
	return WorkloadSimulation{
		ContainerHeight:  4,
		ContainerWidth:   4,
		QueueSize:        1,
		MinBoxHeight:     1,
		MaxBoxHeight:     2,
		MinBoxWidth:      1,
		MaxBoxWidth:      2,
		Iterations:       3,
		AllowBoxRotation: true,
	}
}
