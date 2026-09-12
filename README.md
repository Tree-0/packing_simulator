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
    - `policy/`: Placement policies and container selectors
    - `evaluator/`: Measures policy performance

- `frontend/`: Record simulation snapshots and serve the React visualizer

- `cmd/`
    - `simulate/main.go`: run individual simulations
    - `visualize/main.go`: run and replay simulations/experiments in a browser
    - `batch_sim/main.go`: run experiments across seeds, placement policies, and container selectors, then aggregate results

- `config/`
    - `experiment/`: YAML files for one or more workloads (simulation settings), plus shared seeds, placement policies, container selectors, evaluators, and worker limits
    - `simulate/`: YAML files for individual runs

### Supported Simulation Types
There are a few canonical versions of the [bin packing problem](http://en.wikipedia.org/wiki/Bin_packing_problem). The type of simulation is implicated by a config:
- Objective values:
  - Minimize number of bins to pack all items (`max_containers` omitted, `0`, or `-1`)
  - Maximize objective with limited number of bins (optional config `max_containers`)
- Online/Offline packing:
  - Online: items arrive one at a time, and placement decisions are irreversible (config `queue_size = 1`)
  - Offline: all items are known, and can be rearranged optimally (config `queue_size` equals `iterations`)
  - You can get some intermediate scenario by choosing a `queue_size` in between `1` and `iterations`


### Running simulations

`simulate` loads `config/simulate/config.yml` by default. Command-line flags
override values from that file, so this runs one iteration using the configured
simulation with a different iteration limit:

```sh
go run ./cmd/simulate -iterations 10
```

Use another single-run config with `-config`:

```sh
go run ./cmd/simulate -config config/simulate/config.yml \
  -policy largest-area-bottom-left -container-selector next-k-fit -container-selector-k 3
```

---

`batch_sim` loads `config/experiment/config.yml` by default. Every configuration
uses the same experiment format: it lists one or more named workloads (simulation settings), along
with shared seeds, placement policies, container selectors, evaluators, and
worker count. A one-workload file is the replacement for the old batch
configuration.

```sh
go run ./cmd/batch_sim \
  -config config/experiment/config_standard.yml
```

The runner executes every workload × placement policy × container selector ×
seed combination, prints both individual runs and aggregate means, and can
write CSV output:

```sh
go run ./cmd/batch_sim \
  -config config/experiment/config_rotate_comparison.yml \
  -output-dir cmd/batch_sim/outputs
```

Charts displaying experiment results can be generated with
`cmd/visualize/plot_experiments.ipynb`. The notebook distinguishes placement
policies from container selectors and defaults to the most recent CSV files
generated in `cmd/batch_sim/outputs/`.

#### Config File Layout
There are two similar but separate types of configuration files. Simulation configs
contain one set of simulation settings, to be run once. Experiment configs contain
multiple named simulation settings, along with seeds, placement policies,
container selectors, and evaluators. Combinations of these lists run as
separate simulations, so results can be exported and compared.

Single-run files configure one selector with `container_selector`; experiments
configure a list with `container_selectors`. Omit either setting to use the
backward-compatible `first-fit` default. `next-k-fit` requires a positive `k`:

```yaml
# config/simulate/*.yml
container_selector:
  name: next-k-fit
  k: 3

# config/experiment/*.yml
container_selectors:
  - name: first-fit
  - name: next-k-fit
    k: 3
```

Examples of configs can be found in `config/simulate/` and `config/experiment/` respectively.

[TODO] [Outline meaning of config fields here, or provide a separate file/doc]

---

### Browser visualizer

`visualize` accepts the same single-simulation config and override flags as
`simulate`. It records the run, starts a loopback-only web server, and prints
the URL to open:

```sh
go run ./cmd/visualize
```

The browser view starts at the initial world state and automatically plays one
frame per simulation timestamp. Every allocated container is shown from its
creation onward. Playback can be paused, restarted, stepped, scrubbed, or sped
up and slowed down; the dashboard follows the selected frame and shows the same
run counts and evaluations as the console command.

Use a different config, placement policy, container selector, or listen address with flags:

```sh
go run ./cmd/visualize -config config/simulate/config.yml \
  -policy largest-area-bottom-left -container-selector next-fit -addr 127.0.0.1:9090
```

To replay every workload × placement policy × container selector × seed
combination from an experiment, use the mutually exclusive
`-experiment-config` mode:

```sh
go run ./cmd/visualize \
  -experiment-config config/experiment/config_standard.yml
```

Experiment runs are recorded concurrently using the experiment's `workers`
setting (`0` uses `GOMAXPROCS`) and displayed in configuration order. They
share one playback clock. **Full** shows every run, **Partial** provides a
paginated grid, and **Single** provides the detailed dashboard; selecting a
grid card opens that run in Single view without losing the shared playhead.
Only the evaluators listed in the experiment file appear for those runs.

The visualizer reruns deterministic simulations from YAML and pre-records all
frames before the server starts. It does not load placement histories from
batch CSV output, persist recordings, or stream runs progressively.

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
