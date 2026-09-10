# Policy Decomposition

Need policies to do the following, in order:

- Order batch (queued boxes)
- Choose container to place next box
- Choose placement within container

### Batch Ordering Policies:
- area descending

**An important distinction is that these policies are dealing with what I believe are 1D containers. Need to consider how choosing a 2D placement within a container factors in**

### Container Choice Policies:

#### Online Algorithms
**Some simpler policies:**
- [ ] Next Fit:
    - Keeps one bin open at a time. When next item doesn't fit, close the bin and open a new one.
- [ ] Next-K Fit:
    - Keep the last `k` bins open, choose first bin where item fits.
- [ ] First-Fit:
    - Keep all bins open.
- [ ] Best-Fit:
    - keep all bins open. Attempts to place each new item into the bin with MAXIMUM load (some objective function I will have to define, maybe just utilization?)
- [ ] Worst-Fit
    - Attempts to place each new item into the bin with the MINIMUM load.
- [ ] Almost Worst-Fit
    - Attempts to place each new item inside the second most empty open bin (or emptiest bin if there are two such bins). If does not fit, try emptiest one next.

**More complicated policies:**
- [ ] Refined-first-fit packing
    1. Partitions bins and items into  these four ranges:
        - `(0, 0.33]`, `(0.33, 0.4]`, `(0.4, 0.5]`, `(0.5, 1]`
        - TODO: what do these ranges mean?
    2. Assign item to the first bin in it's matching class with First Fit.
- [ ] Harmonic-k partition packing
    1. Also partitions bins and items into `k-1` parts by size, but with the following method:
        - `(1/(j+1), 1/j] for 1 <= j < k` and `I_k := (0, 1/k]`
- [ ] Refined-harmonic
    1. places items larger than 1/3 using refined-first-fit
    2. Smaller items are placed with harmonic-k
    - The intuition is to reduce the huge waste for bins containing pieces that are just larger than 1/2

#### Offline Algorithms
- First-Fit Decreasing
    - Order the items by descending size, then calls online First Fit
- Next-fit Decreasing
    - Orders items by descending size, then calls online Next Fit
- Modified First-Fit Decreasing
    - Classify items into four size classes ( > 1/2 bin, > 1/3 bin, > 1/6 bin, smaller). 