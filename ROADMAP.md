# Suggested Roadmap (ChatGPT ordering rec)
I would revise the roadmap into this sequence:
1. Formalize the problem, information available to policies, termination rule, and primary objectives.
2. Make workloads identical across policies and produce machine-readable experimental results.
3. Add statistical aggregation, runtime measurements, and multiple workload families.
4. Separate ordering, bin-selection, and within-bin placement strategies.
5. Implement several established 2D heuristics.
6. Add a small-instance exact model and report optimality gaps.
7. Introduce multiple containers and minimize bins used.
8. Add simple structural-support constraints.
9. Profile and experiment with bitsets, skyline, and maximal-empty-rectangle representations.
10. Explore stochastic arrivals/departures, continuous coordinates, local search, or 3D according to which subject interests you most.
The strongest next milestone is therefore not another geometric feature. It is an experiment harness that can answer: “Under this workload, how much quality does this heuristic sacrifice, how much faster is it, and how confident am I in that conclusion?” Once you have that, every subsequent feature becomes an optimization experiment rather than merely an implementation exercise.

# Brainstorming
- [ ] Refactor backend into `evaluator`, `policy`, other packages as needed... they are going to grow

- [ ] Add an option to have multiple containers (fixed config container-count, OR minimize bins needed. TWO DIFFERENT PROBLEMS)
    - Implement heuristics for the more standard version of the bin-packing problem
    - https://en.wikipedia.org/wiki/Bin_packing_problem

- [ ] Learn performance profiling to see how long different parts of the simulation are taking
    -  Use that to inform what you decide to optimize
    - Profile:
        - Time to place each box
        - Candidate positions examined
        - Memory usage
        - Scaling with container area and number of placed boxes

- [ ] Additional evaluators
    - Acceptance rate or accepted area
    - Number of containers used
    - Runtime per decision
    - Number of feasible placements remaining for sampled future boxes
    - regret or optimality gap against an exact solver
    - time-averaged utilization rather than only terminal utilization (how efficient are we throughout the process, not just at the end?)


- [ ] Aggregate results from batch simulations to understand generally how well heuristics perform for each metric
    - [ ] MILP solver to compare heuristics against -> expensive one time run per sim
        - [ ] Define objectives
        - Two types of solvers:
            - [ ] Global optimum for the entire competitior: upper-bound, not a fair competitor
            - [ ] Local optimum for the boxes we have seen at time t in the simulation: better comparison for how our heuristics are doing (implies already placed boxes could be shuffled in the optimum)
        - [ ] long-term: Multi-objective optimization??
    - [ ] Develop a few different types of workloads
        - mostly small boxes
        - mostly large boxes
        - tall or wide boxes
        - bimodal small & large
        - near-square boxes
        - hand-designed adversarial instances
        - correlated dimensions rather than independent width and height
    - Heuristics that are good on some distributions may be bad in others

- [ ] new box generator distributions
    - standard
    - bimodal
    - fixed set

- [ ] More packing constraints besides empty space
    - Structural support (boxes must be supported in a certain way?)
        - EASY: Every box must touch the floor or another box
        - EASY: A minimum fraction of bottom edge must be supported
        - MEDIUM: Center of mass must lie over its supported interval
        - HARD: Boxes have weight and maximum supported load
        - HARD: Load propagates through the stack
    - Boxes have a value and or weight in addition to their size
        - constrained by the size and weight, we try to maximize value
        - reducible to knapsack problem

- [ ] Alternate coordinates?

- [ ] Long Term: How can I optimize my data structures when the grid gets very large?
    - Make sure to do benchmarking/performance profiling first
- [ ] Long Term: 3D packing? -> hard, computationally even more challenging


# Other Goals
- Heuristic ladder: more complex heuristics, compare quality vs. runtime
- Multiobjetive optimization (BANG I thought of ts before gpt suggested it)
    - Balance utilization, rejection rate, stability, runtime, number of bins
- Local search
    - given a finished packing, try swaps, reordering, simulated annealing, tabu search, large-neighborhood search (I will have to look up what all of these mean)
- Actual discrete-event dynamics
    - Currently just turn-by-turn box generation
    - Stochastic interarrival times and box removals
- Sensitivity analysis
    - Sweep queue size, workload distribution, rotation, heuristic parameters
    - Analyze under what conditions an algorithm is most likely to win