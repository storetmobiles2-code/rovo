# Competitor "paved path" review — Zomato, Swiggy, Toing

| | |
|---|---|
| **Date** | 2026-11-12 |
| **Method** | Public sources only (engineering blogs, conference talks, press). No app decompilation or reverse engineering. |
| **Feeds** | RK-55, OQ-P19, OQ-34 in `docs/planning/30-risks-assumptions-decisions.md`; pricing note in `16` §6.5; M3 entry gate in `28` |

## Verdict

rovo is on the paved path for an early-stage, single-city food-delivery build. **No architecture migration is needed.**

- Zomato and Swiggy run hundreds of microservices on a Kafka backbone because of national scale. Swiggy says hundreds of services take part in fulfilling an order, and it has moved from self-managed Kafka to Confluent's managed service.
- Swiggy did not start there. Its own logistics write-up describes a legacy "delivery monolith" that once powered all critical delivery use cases before it was re-architected. rovo's modular monolith matches that earlier stage, with cleaner module boundaries.
- Zomato details (AWS, microservices, Kafka, Redis) come mostly from secondary sources. Treat them as plausible, not confirmed.

## Confirmed: our dispatch timing matches Swiggy's published model

Swiggy's described ETA: assignment delay + max(prep time, rider's trip to the restaurant) + last-mile time. Its **Just-In-Time assignment** aims to have the rider arrive as the food is ready. rovo ruling **R7** (offer at `max(0, prep − approach − buffer)`) is the same pattern.

## Findings to act on

1. **Batching is a pricing lever, not only an optimisation.** Swiggy introduced JIT assignment, next-order assignment and batching while scaling in 2016. Its budget Eco Saver mode reportedly relies on shared fleets and batching to make ₹99 meals viable. rovo V1 is one active delivery per rider. → **OQ-34**: decide on a `trips` entity before M3, while the dispatch module is still unwritten.
2. **Pricing risk in rovo's own market.** Toing is Swiggy's budget app (launched Pune, Sept 2025). It is already in Hyderabad, Vizag and Vijayawada and reportedly in 49 cities. It advertises ₹99 meals, no platform or packaging fees, and free delivery above ₹99, targeting students, Gen Z and value-conscious users. Swiggy's 99 Store is reportedly in 500 cities with free delivery above ₹49. Rapido is entering with flat ₹25/₹50 fees. rovo's defaults would look dearer. → **RK-55, OQ-P19** (decide by M4). Fees are config, so no code change is needed.

## Unverified / limits

- Whether Toing or the 99 Store cover Mahabubnagar today.
- Toing's backend (likely Swiggy's shared platform; undocumented).
- Several dispatch details come from case-study summaries, not Swiggy's primary posts.
- Toing pricing and reach figures come largely from company press releases.

## Sources (accessed 2026-11-12)

- Engineering challenges at Swiggy — https://bytes.swiggy.com/engineering-challenges-at-swiggy-430dea6c86a3
- Re-Architecting Swiggy's logistics systems — https://bytes.swiggy.com/re-architecting-swiggys-logistics-systems-ddf301a29fa0
- Swiggy and Confluent Kafka (Current talk) — https://current.confluent.io/post-conference-videos-2025/powering-real-time-analytics-predictions-and-recommendations-at-swiggy-with-confluent-kafka-bng25
- Swiggy delivery-time prediction (secondary) — https://www.markhub24.com/post/when-every-minute-is-a-brand-promise-swiggy-s-delivery-time-prediction-algorithm
- Swiggy's Delivery Model case study (ICMR) — https://icmrindia.org/casestudies/catalogue/Leadership%20and%20Entrepreneurship/Swiggys_Delivery_Model-Excerpts.htm
- Swiggy ML platform latency (Akka) — https://akka.io/blog/2x-latency-improvement-in-swiggy-ml-and-ai-platform
- Swiggy tech evolution, Dale Vaz (YourStory) — https://yourstory.com/2019/07/foodtech-startup-swiggy-tech-ai-dale-vaz
- Zomato tech stack (secondary) — https://unboxcareer.com/blog/zomato-careers-india-engineering
- Toing launches in Hyderabad (Telangana Today) — https://telanganatoday.com/affordable-food-delivery-app-toing-launches-in-hyderabad
- Toing launches in Vijayawada (Telugu Times) — https://www.telugutimes.net/en/bnews/toing-launches-in-vijayawada-redefining-affordable-food-delivery-349339.html
- What is Toing (LatestLY) — https://www.latestly.com/technology/what-is-toing-key-features-of-swiggys-new-food-delivery-app-offering-meals-at-hyper-local-prices-7112107.html/amp
- Swiggy 99 Store expands to 500 cities (Business Today) — https://www.businesstoday.in/india/story/swiggy-99-store-expands-to-500-cities-what-consumers-can-order-for-rs49-rs99-549044-2026-08-14
- India's convenience economy is getting expensive (Inc42) — https://inc42.com/features/indias-convenience-economy-is-getting-expensive/
