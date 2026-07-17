# Deterministic Cartography of AI Mind Space: A Formal Model of Attribute Selection as a Reproducible Trajectory Through a Discrete-Continuous Choice Space

---

## Abstract

We present a deterministic model for an image-generation system that constructs prompts by selecting records from $N$ finite attribute databases. A seed $s$ and a series $\sigma$ together drive a family of deterministic hash functions, each producing a 24-bit seed chunk that is scaled by the cardinality of the series-filtered database to select a record. The resulting attribute vector is interpreted as a point in a series-dependent discrete product space $\mathcal{C}_\sigma$, and the sequence of partial selections is interpreted as a path through that space. We show how to render that path as a folded chain in $\mathbb{R}^3$, where each attribute contributes one bond whose length is normalized by its database size and whose direction is mapped deterministically from the same seed chunk that selected the attribute. The central result is a determinism theorem: under a fixed series, fixed databases for that series, fixed hash functions, and a fixed spherical mapping, every $(s, \sigma)$ pair generates a unique trajectory and a unique rendered map. The analogy to protein folding arises because successive bonds point in unrelated directions determined by the seed.

---

## 1. Introduction

Consider an image-generation application that builds a prompt by sampling one record from each of $N$ thematic databases: nouns, adjectives, artistic styles, color palettes, and so on. The user first selects a **series** $\sigma$, which filters the available records in each database, and then provides a **seed** phrase $s$. If the sampling mechanism is a deterministic function of the pair $(s, \sigma)$, then the entire prompt is a deterministic function of that pair. This paper formalizes that observation and develops a geometric interpretation of the selection process.

The intuition we pursue is that each seed defines a walk through a high-dimensional space of attribute combinations. Rather than collapsing the entire walk onto a single 3-D point, we render the walk itself as a folded chain in three dimensions. Each attribute contributes one bond to the chain. Because every step is computable from the seed, the resulting shape is reproducible. When later selections depend on earlier ones, the chain folds back on itself, producing a structure reminiscent of a protein backbone.

---

## 2. The Attribute Selection System

Let the $N$ attribute databases be finite, ordered sets:

$$D_n = \{ d_{n,1}, d_{n,2}, \ldots, d_{n,R_n} \}, \qquad n = 1, 2, \ldots, N$$

where $R_n = |D_n| \geq 1$ is the cardinality of database $n$. Let $S$ denote the seed-phrase space and $\Sigma$ denote the series space. A deterministic hash of the pair $(s, \sigma)$ is split into $N$ fixed-size **seed chunks** $h_1(s, \sigma), \ldots, h_N(s, \sigma)$, each an integer in $\{0, 1, \ldots, 2^{24}-1\}$. For each database $D_n$, define a deterministic selection function:

$$H_n : S \times \Sigma \to \{0, 1, \ldots, 2^{24}-1\}, \qquad H_n(s, \sigma) = h_n(s, \sigma)$$

The selected record index is obtained by scaling the chunk by the series-filtered database size and taking the floor. Let $R_n^{(\sigma)} = |D_n^{(\sigma)}|$ denote the cardinality of database $n$ after applying the series filter $\sigma$:

$$I_n(s, \sigma) = \left\lfloor R_n^{(\sigma)} \cdot \frac{H_n(s, \sigma)}{2^{24}} \right\rfloor$$

with $0 \leq I_n(s, \sigma) \leq R_n^{(\sigma)} - 1$. The selected attribute from database $n$ under series $\sigma$ is then:

$$a_n(s, \sigma) = d_{n,\, I_n(s, \sigma) + 1}^{(\sigma)} \in D_n^{(\sigma)}$$

and the complete attribute vector for the pair $(s, \sigma)$ is:

$$A(s, \sigma) = \bigl( a_1(s, \sigma), a_2(s, \sigma), \ldots, a_N(s, \sigma) \bigr)$$

Thus the generator is a function:

$$G : S \times \Sigma \to \mathcal{C}_\sigma, \qquad G(s, \sigma) = A(s, \sigma)$$

where the **series-dependent choice space** is the Cartesian product:

$$\mathcal{C}_\sigma = D_1^{(\sigma)} \times D_2^{(\sigma)} \times \cdots \times D_N^{(\sigma)}$$

The total number of possible attribute combinations for series $\sigma$ is:

$$|\mathcal{C}_\sigma| = \prod_{n=1}^{N} R_n^{(\sigma)}$$

[[IMG:images/fig1-attribute-selection.png|Figure 1: Seed and series select one record from each filtered database.]]

---

## 3. History as a Trajectory

The order in which databases are consulted is significant. Define the **partial history** after the first $k$ selections under series $\sigma$ as:

$$A_k(s, \sigma) = \bigl( a_1(s, \sigma), a_2(s, \sigma), \ldots, a_k(s, \sigma) \bigr), \qquad 1 \leq k \leq N$$

with $A_0(s, \sigma) = \varnothing$ and $A_N(s, \sigma) = A(s, \sigma)$. Each partial history $A_k(s, \sigma)$ is a vertex in the partial product lattice $D_1^{(\sigma)} \times \cdots \times D_k^{(\sigma)}$. The sequence:

$$A_0(s, \sigma), A_1(s, \sigma), A_2(s, \sigma), \ldots, A_N(s, \sigma)$$

is a path of length $N$ through $\mathcal{C}_\sigma$, moving one dimension at a time.

If each $H_n$ depends only on $(s, \sigma)$, the trajectory is an orthogonal walk in $\mathcal{C}_\sigma$: at step $k$, the path moves parallel to the $k$-th coordinate axis. If $H_n$ depends on $(s, \sigma)$ and the prior history $A_{n-1}(s, \sigma)$, the walk becomes adaptive and may fold through the space. In Section 4 we render either kind of walk as a folded chain in $\mathbb{R}^3$, with each attribute contributing one bond.

---

## 4. Local Frames and the Attribute Chain

To visualize the trajectory, we first normalize each database coordinate so that every attribute contributes a comparable bond length regardless of how large its series-filtered database is. Define the **normalized coordinate** for database $n$ under series $\sigma$ as:

$$u_n(s, \sigma) = \frac{I_n(s, \sigma) + 1}{R_n^{(\sigma)}} \in (0, 1]$$

The value $u_n(s, \sigma)$ measures how far into the series-filtered database $n$ the pair $(s, \sigma)$ reaches: $1$ selects the last record, and values near $0$ select records near the beginning. Because $I_n(s, \sigma) = \lfloor R_n^{(\sigma)} H_n(s, \sigma) / 2^{24} \rfloor$, the normalized coordinate is approximately the chunk fraction $H_n(s, \sigma)/2^{24}$, quantized to one of $R_n^{(\sigma)}$ evenly spaced steps.

Rather than embedding the full attribute vector into $\mathbb{R}^N$, we build a 3-D chain one bond at a time. At step $n$ the current endpoint is $p_{n-1}(s, \sigma) \in \mathbb{R}^3$, and we add a bond whose length is $u_n(s, \sigma)$. The bond direction $d_n(s, \sigma)$ is a unit vector in $\mathbb{R}^3$ that is deterministically derived from the same seed chunk $c_n(s, \sigma)$ used to select the attribute.

Split the seed chunk into two independent 12-bit quantities:

$$c_n(s, \sigma) = c_{n,\theta} + 2^{12} \, c_{n,\phi}, \qquad c_{n,\theta}, c_{n,\phi} \in \{0, 1, \ldots, 2^{12}-1\}$$

and map them to spherical coordinates:

$$\theta_n = 2\pi \, \frac{c_{n,\theta}}{2^{12}}, \qquad \phi_n = \arccos\!\left(1 - 2 \, \frac{c_{n,\phi}}{2^{12}}\right)$$

The bond direction is then:

$$d_n(s, \sigma) = \bigl(\sin\phi_n \cos\theta_n,\; \sin\phi_n \sin\theta_n,\; \cos\phi_n\bigr)$$

This mapping distributes directions uniformly over the unit sphere. Because the input chunk $c_n$ is itself a deterministic function of $(s, \sigma)$, the direction is reproducible; because the chunks vary from step to step, successive bonds point in unrelated directions, producing the protein-folding effect.

The chain is defined recursively by $p_0(s, \sigma) = 0$ and:

$$p_n(s, \sigma) = p_{n-1}(s, \sigma) + u_n(s, \sigma) \, d_n(s, \sigma), \qquad n = 1, 2, \ldots, N$$

Thus the $n$-th bond reaches from $p_{n-1}(s, \sigma)$ to $p_n(s, \sigma)$, its length encodes the normalized selection $u_n(s, \sigma)$, and its direction is a deterministic function of the same seed chunk that selected the attribute. Every attribute is represented by exactly one bond.

---

## 5. The Protein-Folding Map in Three Dimensions

The sequence of endpoints:

$$p_0(s, \sigma), p_1(s, \sigma), p_2(s, \sigma), \ldots, p_N(s, \sigma)$$

is a piecewise-linear chain in $\mathbb{R}^3$. The full chain is the **protein-folding map** of the pair $(s, \sigma)$: a deterministic spatial object whose geometry records the entire history of attribute selections for that series.

Each bond can also be expressed in spherical coordinates relative to its starting endpoint. The $n$-th bond has length $u_n(s, \sigma)$ and direction $d_n(s, \sigma)$. If we write $d_n(s, \sigma) = (\sin\phi_n \cos\theta_n,\; \sin\phi_n \sin\theta_n,\; \cos\phi_n)$, then the bond is the spherical vector:

$$(u_n(s, \sigma), \theta_n, \phi_n)$$

where the angles $\theta_n$ and $\phi_n$ come from the deterministic spherical mapping of the seed chunk $c_n(s, \sigma)$. The pair $(s, \sigma)$ determines both the radial reach $u_n(s, \sigma)$ and the direction $d_n(s, \sigma)$; successive bonds point in unrelated directions, so the chain folds.

[[IMG:images/fig2-protein-fold.png|Figure 2: Each normalized attribute becomes one bond of a 3-D chain.]]

Two pairs $(s, \sigma)$ and $(s', \sigma')$ produce the same final image only if $\sigma = \sigma'$ and $A(s, \sigma) = A(s', \sigma')$. They produce similar folding maps if they share the same series and their normalized coordinate sequences $\{u_n(s, \sigma)\}$ and $\{u_n(s', \sigma)\}$ are close, because the spherical mapping is shared. Under a richer embedding — for example, replacing $u_n(s, \sigma)$ with a learned feature distance $\|\phi_n(a_n(s, \sigma)) - \phi_n(a_n(s', \sigma))\|$ — nearby records in a database cause nearby bond lengths, and semantically similar prompts produce visually similar folds.

### 5.3 Rendering the Folding Map

Figure 2 is a deterministic rendering of the chain for an illustrative choice of parameters. The rendering uses the same seed-chunk mapping that governs the dalledress image generator; only the seed chunks themselves are chosen as a representative deterministic sequence rather than derived from a specific seed and series.

For the figure we use $N = 24$ bonds. Each synthetic seed chunk $c_n$ is produced by a deterministic pseudo-random function of $n$; in the live system these chunks come directly from the hash of $(s, \sigma)$. From each chunk we compute a normalized length $u_n$ and a direction $d_n$ exactly as in Section 4:

$$u_n = \frac{(c_n \bmod 1000) + 1}{1000}, \qquad c_n = c_{n,\theta} + 2^{12} c_{n,\phi}$$

$$\theta_n = 2\pi \frac{c_{n,\theta}}{2^{12}}, \qquad \phi_n = \arccos\!\left(1 - 2 \frac{c_{n,\phi}}{2^{12}}\right)$$

$$d_n = (\sin\phi_n \cos\theta_n,\; \sin\phi_n \sin\theta_n,\; \cos\phi_n)$$

The endpoints are then:

$$p_n = p_{n-1} + u_n \, d_n, \qquad n = 1, \ldots, N$$

with $p_0 = 0$. For the 2-D illustration, each 3-D point $v = (v_x, v_y, v_z)$ is projected by perspective projection with camera distance $d = 5.0$:

$$(v_x', v_y') = \left( \frac{v_x}{v_z + d},\; \frac{v_y}{v_z + d} \right)$$

The foldvis renderer implements this projection literally, then scales and translates the result to fit the image canvas. Bonds are drawn back-to-front by their average projected depth, so nearer bonds occlude farther bonds and the fold reads as a single coherent object.

---

## 6. Spherical History of the Endpoint

The final endpoint $p_N(s, \sigma)$ can itself be described in spherical coordinates. Its distance from the origin is:

$$r(s, \sigma) = \|p_N(s, \sigma)\| = \left\| \sum_{n=1}^{N} u_n(s, \sigma) \, d_n(s, \sigma) \right\|$$

and its direction on the unit sphere is $\hat{p}_N(s, \sigma) = p_N(s, \sigma) / r(s, \sigma)$. Writing $\hat{p}_N(s, \sigma)$ in standard spherical coordinates $(r(s, \sigma), \Theta(s, \sigma), \Phi(s, \sigma))$ gives a compact summary: the radius measures the total cumulative reach of all attribute selections within the series, while the angles record the net orientation produced by the randomly-oriented bond directions.

This endpoint summary is lossy — it discards the path geometry — but it is deterministic and provides a single 3-D coordinate for the generated image.

---

## 7. Determinism

**Theorem 1 (Deterministic Cartography).** Given:

1. A fixed series $\sigma \in \Sigma$;
2. Fixed databases $D_1^{(\sigma)}, \ldots, D_N^{(\sigma)}$ for that series with fixed orderings;
3. Fixed deterministic hash functions $H_1, \ldots, H_N$;
4. A fixed deterministic spherical mapping $\psi : \mathbb{Z}_{2^{24}} \to S^2$;

then for every seed $s \in S$ the attribute vector $A(s, \sigma)$, the normalized coordinates $\{u_n(s, \sigma)\}_{n=1}^{N}$, the 3-D chain $\{p_n(s, \sigma)\}_{n=0}^{N}$, and the endpoint spherical coordinates $(r(s, \sigma), \Theta(s, \sigma), \Phi(s, \sigma))$ are uniquely determined.

**Proof.** For each $n$, the index $I_n(s, \sigma) = \bigl\lfloor R_n^{(\sigma)} H_n(s, \sigma) / 2^{24} \bigr\rfloor$ is uniquely determined by $H_n(s, \sigma)$ and $R_n^{(\sigma)}$, hence the normalized coordinate $u_n(s, \sigma) = \bigl(I_n(s, \sigma) + 1\bigr) / R_n^{(\sigma)}$ is determined. The seed chunk $c_n(s, \sigma)$ is determined by the hash, and the fixed spherical mapping $\psi$ determines the bond direction $d_n(s, \sigma)$. Therefore each bond vector $u_n(s, \sigma) \, d_n(s, \sigma)$ is determined, and by induction the entire chain $\{p_n(s, \sigma)\}$ is determined. The endpoint $p_N(s, \sigma)$ and its spherical coordinates are continuous functions of the chain, hence determined. ∎

---

## 8. Discussion and Caveats

The phrase "the mind of the AI" should be understood as a productive metaphor rather than a literal claim. The model developed here describes the prompt vocabulary space — the discrete set of symbolic attributes from which prompts are assembled. A complete theory would also relate these attribute vectors to the latent diffusion space or activation patterns of the image model itself.

Several limitations deserve emphasis:

- **Hash collisions and bias.** Determinism does not imply uniformity. If $H_n$ has poor mixing properties, certain records may be selected far more often than others, distorting the bond-length distribution of the map.
- **Spherical mapping choice.** The visual fold depends on the fixed spherical mapping $\psi$. A different deterministic mapping produces a different but equally valid rendering of the same deterministic trajectory.
- **Information loss in the endpoint summary.** The spherical coordinates $(r(s, \sigma), \Theta(s, \sigma), \Phi(s, \sigma))$ of the final endpoint discard the path geometry; distinct chains may end at the same point.
- **Adaptive versus non-adaptive selection.** The protein-folding analogy is strongest when $H_n$ depends on prior selections. If every $H_n$ depends only on $(s, \sigma)$, the walk in $\mathcal{C}_\sigma$ is an orthogonal walk, but the 3-D chain still folds because successive bond directions are unrelated.
- **Invertibility.** The generator $G : S \times \Sigma \to \mathcal{C}_\sigma$ is deterministic but need not be injective or surjective for a fixed series. Multiple seeds may map to the same attribute vector within a series, and some vectors may be unreachable.
- **Cross-series comparison.** Two different series define different choice spaces $\mathcal{C}_\sigma$ and $\mathcal{C}_{\sigma'}$. A seed $s$ in series $\sigma$ and the same seed phrase in series $\sigma'$ are different points in the generative system and generally produce different images and different folding maps.

---

## 9. Conclusion

We have shown that a seed-driven, series-contextualized attribute-selection system defines a deterministic trajectory through a finite product space. By rendering that trajectory as a folded chain in $\mathbb{R}^3$ — one bond per attribute, with length normalized by the series-filtered database size and direction mapped deterministically from the same seed chunk — the abstract choice history becomes a reproducible geometric object. The framework justifies the intuition that each generated image can be assigned a unique coordinate history within its series and, under suitable conditions, a unique folded path through the space of possible prompts.

---

## References

No external references were used in the preparation of this note.
