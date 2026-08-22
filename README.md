# Packing Simulator

Goal:

Incoming queue of rectangular boxes of various dimensions, must be packed into a grid.

Several heuristic and optimization policies evaluated with a variety of objective functions.

Run randomized simulations over time, observe how each policy performs with each metric.

### Structure

- `backend/`
    - `simulator`: Owns time and event processing
    - `world`: The simulation model space - grid, placed boxes, queue
    - `generator`: Creates box-arrival events
    - `policy_*`: Different packing policies
    - `evaluator_*`: Measures policy performance

- `frontend/`: Observe and display state

- `cmd/`
    - `simulate/main.go`: run individual simulations
    - `batch_sim/main.go`: run groups of simulations for a set of seeds and policies, collect results

- `config/`
    - `batch_sim/`: YAML files for batch runs, including seeds, policies, evaluators, and workers
    - `simulate/`: YAML files for individual runs

### Running simulations

`simulate` loads `config/simulate/config.yml` by default. Command-line flags
override values from that file, so this runs one iteration using the configured
simulation with a different iteration limit:

```sh
go run ./cmd/simulate -iterations 1
```

Use another single-run config with `-config`:

```sh
go run ./cmd/simulate -config config/simulate/config.yml -policy largest-area-bottom-left
```

`batch_sim` loads `config/batch_sim/config.yml` by default. Its YAML format is
different because it specifies multiple seeds, policies, and evaluators. A batch simulation
runs (number of seeds) * (number of policies) simulations to try all pairwise combinations,
and applies all evaluators to each.
