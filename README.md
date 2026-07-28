# Packing Simulator

Goal:

Incoming queue of rectangular boxes of various dimensions, must be packed into a grid.

Several heuristic and optimization policies evaluated with a variety of objective functions.

Run randomized simulations over time, observe how each policy performs with each metric.

### Structure

- `backend/`
    - `simulator.go`: Owns time and event processing
    - `world.go`: The simulation model space - grid, placed boxes, queue
    - `generator.go`: Creates box-arrival events

    - `policy/`: Different packing policies

    - `evaluator`: measures policy performance

- `frontend/`: Observe and display state