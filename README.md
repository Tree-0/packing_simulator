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

- `frontend/`: Record simulation snapshots and serve the React visualizer

- `cmd/`
    - `simulate/main.go`: run individual simulations
    - `visualize/main.go`: run and replay an individual simulation in a browser
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

### Browser visualizer

`visualize` accepts the same single-simulation config and override flags as
`simulate`. It records the run, starts a loopback-only web server, and prints
the URL to open:

```sh
go run ./cmd/visualize
```

The browser view starts at the empty container and automatically plays one
frame per simulation timestamp. Playback can be paused, restarted, stepped,
scrubbed, or sped up and slowed down. The dashboard follows the selected frame
and shows the same run counts and evaluations as the console command.

Use a different config, policy, or listen address with flags:

```sh
go run ./cmd/visualize -config config/simulate/config.yml \
  -policy largest-area-bottom-left -addr 127.0.0.1:9090
```

The production React bundle is checked in and embedded in the Go binary, so
running the visualizer only requires Go.

### Frontend development

Changing the React source requires Node 24 and npm. On macOS with Homebrew:

```sh
brew install node@24
cd frontend/web
npm ci
```

For hot reload, run `go run ./cmd/visualize` in one terminal and `npm run dev`
from `frontend/web` in another. Vite serves the development UI at
`http://127.0.0.1:5173` and proxies its API calls to the Go server.

Before committing frontend changes, run:

```sh
cd frontend/web
npm run check
```

This type-checks and tests the components, then rebuilds the embedded `dist`
assets. Commit the updated bundle along with the React source.

`batch_sim` loads `config/batch_sim/config.yml` by default. Its YAML format is
different because it specifies multiple seeds, policies, and evaluators. A batch simulation
runs (number of seeds) * (number of policies) simulations to try all pairwise combinations,
and applies all evaluators to each.
