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
    - `*.yml`: files used to specify batch simulation runs