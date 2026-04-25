# TODO list

## État des lieux au 2026-04-24

- Monorepo avec trois surfaces principales: `front-vue`, `back-go`, `back-kotlin`.
- Le frontend Vue 3 couvre déjà dashboard, charts, heatmap, statistiques, badges, activités, carte, segments, détail activité et routes.
- Le backend Go suit une architecture plus découpée par use cases et reste important pour les builds binaires.
- Le backend Kotlin reste riche côté services métier, fournisseurs Strava/GPX/FIT, SRTM et Spring Boot.
- Les deux backends exposent un contrat `/api/...` proche et partagent des fixtures de parité pour les routes.
- Le moteur Routes est la zone la plus sensible: OSRM, target/shape generation, anti-retrace, diagnostics, export GPX, parité Go/Kotlin.
- La documentation route est déjà solide (`docs/route-generation-engine.md`, guides de checks manuels OSRM).
- Les tests backend sont nombreux. Le frontend a quelques tests Vitest ciblés, mais peu de couverture composant/parcours utilisateur.
- Plusieurs fichiers de packaging et docs semblent désalignés avec les versions réellement déclarées dans les manifests.

## Garde-fous permanents

- Garder Go et Kotlin alignés pour tout changement de génération de routes.
- Ne jamais transformer l'historique en pénalité de nouveauté: il doit rester un signal positif de corridors connus.
- Préserver les règles anti-retrace strictes hors zone départ/arrivée.
- Garder le comportement de zone départ/arrivée 2 km explicite et testé.
- Préserver `X-Request-Id` et les diagnostics exploitables sur les endpoints de génération.
- Ne pas changer silencieusement les contrats API: ajouter migration, compatibilité ou tests de contrat.

## Améliorations techniques proposées

### Priorité haute

- [x] `TECH-P0-01` (`P0`, `M`) - Aligner les toolchains et les builds Docker.
  Owners: `Infra`, `Back-Go`, `Back-Kotlin`, `Front`.
  Constat initial:
  - `back-go/go.mod` déclarait Go `1.26.2`, mais `back-go/Dockerfile` buildait avec `golang:1.24.3-alpine`.
  - `back-kotlin/build.gradle.kts` demandait Java `25`, mais `back-kotlin/Dockerfile` utilisait JDK/OpenJDK `23`.
  - `front-vue/package.json` demandait Node `>=25.9.0`, mais le Dockerfile utilisait `node:lts-alpine`.
  - `back-kotlin/README.md` mentionnait encore JDK 21.
  Scope:
  - harmoniser Dockerfiles, CI, README et scripts locaux sur les mêmes versions,
  - ajouter une section "versions supportées" dans `docs/README.md`,
  - faire échouer la CI si un Docker build ne respecte pas les manifests.
  Acceptance:
  - `docker compose` build Go/Kotlin/Front sans contournement de version,
  - les README ne donnent plus de consignes contradictoires.
  Statut 2026-04-24:
  - Dockerfiles alignés sur Go `1.26.2`, Java `25`, Gradle `9.4.1`, Node `25.9.0`,
  - workflows CI/build manuel et scripts de build Go alignés sur les mêmes versions,
  - section versions supportées ajoutée dans `docs/README.md`,
  - check CI `scripts/check-toolchains.sh` ajouté pour détecter les dérives manifest/Docker/CI.

- [x] `TECH-P0-02` (`P0`, `M`) - Rendre les modes Docker réellement exécutables de bout en bout.
  Owners: `Infra`, `Front`, `Back-Go`, `Back-Kotlin`.
  Constat:
  - le Nginx frontend sert l'app statique mais ne proxy pas `/api`,
  - les compose Go/Kotlin ne connectent pas clairement `front` et `back` sur le même chemin réseau applicatif,
  - le serveur Go écoute sur `localhost:<port>`, ce qui est fragile en conteneur.
  Scope:
  - choisir un mode officiel: backend qui sert le frontend, ou frontend Nginx qui proxy `/api` vers `back:8080`,
  - rendre l'adresse d'écoute backend configurable (`0.0.0.0` en conteneur),
  - ajouter healthchecks backend/front/OSRM et un test smoke Docker.
  Acceptance:
  - après `docker compose up`, l'UI chargée depuis le conteneur appelle `/api/health/details` avec succès,
  - le même scénario fonctionne pour Go et Kotlin.
  Statut 2026-04-24:
  - mode officiel retenu: frontend Nginx qui proxy `/api/...` vers le service backend `back:8080`,
  - compose Go/Kotlin alignés: `front` et `back` partagent le même réseau applicatif, le cache Strava a un fallback local, les backends écoutent sur `0.0.0.0` en conteneur,
  - healthchecks ajoutés pour backend, frontend et OSRM optionnel; les images backend embarquent `curl`,
  - OSRM optionnel raccordé au réseau compose via [docker-compose-routing-osrm.yml](/Users/nicolas/Workspace/mystravastats-2/docker-compose-routing-osrm.yml),
  - smoke script ajouté: `./scripts/smoke-docker-compose.sh go|kotlin`.

- [x] `TECH-P0-03` (`P0`, `S`) - Corriger les badges multi-activity-types.
  Owners: `Back-Go`, `Back-Kotlin`.
  Constat:
  - TODO existants dans `BadgesService.kt`, `BadgeCheckResultDto.kt` et `converters.go`,
  - le type du badge dépend du premier sport sélectionné, ce qui peut devenir faux avec les sélections multi-sports du header.
  Scope:
  - définir le contrat attendu pour `Ride+Gravel+MTB`, `Run+TrailRun`, `Hike+Walk`,
  - produire des badges agrégés stables ou des badges par famille de sport,
  - aligner DTO Go/Kotlin et tests.
  Acceptance:
  - plus de TODO multi-sports dans la logique badges,
  - résultats déterministes quand plusieurs sports sont sélectionnés.
  Statut 2026-04-24:
  - contrat retenu: badges agrégés par famille sportive stable,
  - `Ride`, `GravelRide`, `MountainBikeRide`, `VirtualRide` et `Commute` produisent des types DTO `Ride*Badge`,
  - `Run` et `TrailRun` produisent des types DTO `Run*Badge`,
  - `Hike` et `Walk` produisent des types DTO `Hike*Badge`,
  - Go/Kotlin partagent la même résolution de famille et les tests couvrent les sélections multi-types.

### Priorité moyenne

- [ ] `TECH-P1-01` (`P1`, `L`) - Mettre le contrat API sous contrôle OpenAPI partagé.
  Owners: `Back-Kotlin`, `Back-Go`, `Front`, `QA`.
  Constat:
  - Springdoc existe côté Kotlin, Swagger existe côté Go, mais le frontend maintient ses interfaces à la main.
  Scope:
  - choisir une source de vérité OpenAPI,
  - générer les types TypeScript et éventuellement un client API typé,
  - ajouter des tests de conformité Go/Kotlin sur les DTO sensibles (`routes`, `statistics`, `dashboard`, `activities`).
  Acceptance:
  - une divergence de champ ou d'enum casse la CI avant d'arriver dans l'UI.

- [ ] `TECH-P1-02` (`P1`, `M`) - Automatiser les checks routes aujourd'hui manuels.
  Owners: `QA`, `Back-Go`, `Back-Kotlin`, `Front`.
  Constat:
  - les docs de validation OSRM sont précises, mais plusieurs validations restent manuelles.
  Scope:
  - transformer les scénarios anti-retrace, direction, surface, fallback et shape tuning en smoke tests automatisés,
  - lancer ces checks uniquement derrière profil CI/local OSRM pour éviter de ralentir la CI standard,
  - capturer les diagnostics clés en artifact.
  Acceptance:
  - un changement route peut être validé avec une commande unique,
  - les cas terrain critiques restent reproductibles.

- [ ] `TECH-P1-03` (`P1`, `M`) - Étendre la couverture frontend.
  Owners: `Front`, `QA`.
  Constat:
  - les tests Vitest couvrent surtout stores/routes/charts utils,
  - les parcours UI riches (filtres, détail activité, route map, import GPX, erreurs API) sont peu protégés.
  Scope:
  - ajouter tests composants pour `HeaderBar`, `RoutesView`, `ActivityHeatmapChart`, `HeartRateZoneAnalysisPanel`,
  - ajouter quelques tests e2e/smoke avec backend mocké ou fixtures,
  - vérifier les états loading/erreur/cache.
  Acceptance:
  - les workflows utilisateurs principaux sont protégés sans dépendre de Strava.

- [x] `TECH-P1-04` (`P1`, `M`) - Aligner le support FIT Go/Kotlin.
  Owners: `Back-Go`, `Back-Kotlin`.
  Constat:
  - le Go reconstruit un stream power à partir des records FIT,
  - le Kotlin lit `avgPower` mais garde un TODO "Calculate ?" quand la session ne le fournit pas.
  Scope:
  - calculer `averageWatts`, `weightedAverageWatts` et kilojoules depuis le stream quand les champs session sont absents,
  - partager des fixtures FIT minimales ou des tests de mapping,
  - documenter les limites des champs FIT selon appareils.
  Acceptance:
  - les métriques de puissance FIT ne tombent plus silencieusement à zéro quand le stream permet de les calculer.
  Statut 2026-04-24:
  - Go et Kotlin appliquent la même règle: `avgPower` session reste prioritaire, sinon le stream `record.power` calcule `averageWatts`, `weightedAverageWatts` et kilojoules,
  - la moyenne inclut les échantillons à zéro, ignore les valeurs invalides/négatives et ne s'active que si au moins un échantillon positif existe,
  - `weightedAverageWatts` utilise une approximation de puissance normalisée par fenêtre glissante de 30 échantillons, avec fallback sur la moyenne pour les streams courts,
  - les limites FIT sont documentées dans [docs/README.md](/Users/nicolas/Workspace/mystravastats-2/docs/README.md),
  - des tests de mapping couvrent le fallback stream, la priorité session et les streams sans puissance exploitable côté Go/Kotlin.

- [ ] `TECH-P1-05` (`P1`, `L`) - Réduire le risque de divergence Go/Kotlin hors routes.
  Owners: `Back-Go`, `Back-Kotlin`, `QA`.
  Scope:
  - ajouter des fixtures partagées pour statistiques, badges, dashboard, heatmap et activités détaillées,
  - comparer au minimum les champs agrégés et les cas limites de dates/streams manquants,
  - documenter les divergences acceptées quand une fonctionnalité n'existe que dans un backend.
  Acceptance:
  - la parité critique n'est plus limitée au moteur routes.

### Priorité basse

- [ ] `TECH-P2-01` (`P2`, `M`) - Nettoyer la stratégie d'assets frontend embarqués.
  Owners: `Front`, `Back-Kotlin`, `Back-Go`, `Infra`.
  Constat:
  - Kotlin contient des assets compilés dans `src/main/resources/static`,
  - Go embarque `public`,
  - le frontend a son propre build Vite.
  Scope:
  - définir si les assets compilés sont générés au build ou versionnés,
  - éviter les assets obsolètes dans les backends,
  - rendre les scripts de capture docs compatibles avec le mode retenu.
  Acceptance:
  - un build release ne peut pas embarquer une ancienne UI par accident.

- [x] `TECH-P2-02` (`P2`, `S`) - Centraliser la configuration runtime.
  Owners: `Back-Go`, `Back-Kotlin`, `Docs`.
  Scope:
  - lister les variables `STRAVA_CACHE_PATH`, `FIT_FILES_PATH`, `GPX_FILES_PATH`, `OSM_ROUTING_*`, `CORS_ALLOWED_ORIGINS`,
  - exposer les valeurs effectives non sensibles dans `/api/health/details`,
  - documenter les valeurs par défaut Go/Kotlin au même endroit.
  Acceptance:
  - diagnostiquer une mauvaise config ne nécessite plus de lire plusieurs fichiers.
  Statut 2026-04-24:
  - configuration runtime centralisée côté Go/Kotlin avec exposition non sensible sous `runtimeConfig` dans `/api/health/details`,
  - CORS Kotlin raccordé à `CORS_ALLOWED_ORIGINS` comme Go,
  - page Diagnostics enrichie avec une section `Runtime Config`,
  - table unique des variables/defaults Go/Kotlin ajoutée dans `docs/README.md`.

- [x] `TECH-P2-03` (`P2`, `S`) - Durcir la gestion CORS et credentials.
  Owners: `Back-Go`, `Back-Kotlin`, `Infra`.
  Scope:
  - harmoniser la configuration CORS Go/Kotlin,
  - rendre les origins configurables côté Kotlin comme côté Go,
  - ajouter tests de préflight OPTIONS.
  Acceptance:
  - comportement identique en dev, Docker et release locale.
  Statut 2026-04-24:
  - configuration CORS Go/Kotlin harmonisée: origins explicites via `CORS_ALLOWED_ORIGINS`, credentials activés, méthodes `GET/POST/PUT/DELETE/OPTIONS`,
  - header `X-Request-Id` autorisé avec `Content-Type` et `Authorization` pour préserver les diagnostics côté routes,
  - tests de préflight OPTIONS ajoutés côté Go et Kotlin avec rejet d'une origin non configurée,
  - `/api/health/details.runtimeConfig.cors` expose désormais origins, méthodes, headers et credentials.

## Améliorations fonctionnelles proposées

### Priorité haute

- [x] `FUNC-P0-01` (`P0`, `L`) - Objectifs annuels et projections.
  Owners: `Product`, `Front`, `Back-Go`, `Back-Kotlin`.
  Proposition:
  - permettre de définir des objectifs par sport: distance, dénivelé, temps, nombre de sorties, jours actifs, Eddington,
  - afficher progression, rythme nécessaire, projection fin d'année et statut `en avance / juste / en retard`,
  - stocker les objectifs localement dans le cache athlète.
  Valeur:
  - transforme le dashboard historique en tableau de bord d'aide à la décision.
  Acceptance:
  - objectifs persistés sans dépendance à Strava,
  - affichage cohérent avec le filtre sport/année courant.
  Statut 2026-04-25:
  - endpoint Go/Kotlin `GET/PUT /api/dashboard/annual-goals` ajouté,
  - objectifs distance, dénivelé, temps, sorties, jours actifs et Eddington persistés dans le cache athlète,
  - panneau dashboard ajouté avec progression, projection fin d'année, rythme requis et statut.

- [x] `FUNC-P0-03` (`P0`, `M`) - Page Diagnostics utilisateur.
  Owners: `Product`, `Front`, `Back-Go`, `Back-Kotlin`.
  Proposition:
  - afficher cache, provider actif, années disponibles, état warmup, quota/rate-limit Strava connu, santé OSRM,
  - ajouter actions guidées: relancer un warmup, vérifier OSRM, ouvrir le dossier cache, expliquer un mode dégradé.
  Valeur:
  - réduit fortement le temps passé à comprendre pourquoi une vue est vide ou lente.
  Acceptance:
  - `/api/health/details` alimente une vue lisible sans ouvrir les logs.
  Statut 2026-04-24:
  - page `/diagnostics` ajoutée avec synthèse provider/cache/années, warmup, rate-limit Strava, santé OSRM, fichiers cache et payload brut,
  - actions UI ajoutées: refresh global, check OSRM via refresh santé et copie du chemin cache,
  - mode dégradé traduit en raisons utilisateur: backend indisponible, rate-limit, OSRM down/misconfigured/disabled, warmup ou refresh en cours,
  - `/api/health/details` enrichi côté Go/Kotlin avec provider actif, nombre d'activités, années disponibles et état des jobs background,
  - les providers locaux Kotlin GPX/FIT exposent désormais aussi leurs diagnostics de base,
  - onglet renommé visuellement en `Status`, avec section cache enrichie: source, activités, années, fichiers, taille, dates manifest et warmup.

### Priorité moyenne

- [ ] `FUNC-P1-03` (`P1`, `M`) - Analyse matériel.
  Owners: `Product`, `Stats`, `Front`.
  Proposition:
  - exploiter les données gear/bike/shoe déjà présentes dans les modèles Strava,
  - afficher distance, temps, D+, vitesse moyenne, records et maintenance par équipement,
  - gérer les équipements absents pour GPX/FIT.
  Acceptance:
  - vue filtrable par vélo/chaussures avec totaux fiables.

- [ ] `FUNC-P1-04` (`P1`, `M`) - Comparaison d'activité à effort similaire.
  Owners: `Product`, `Stats`, `Front`.
  Proposition:
  - dans le détail activité, comparer avec les sorties proches en distance/D+/sport/saison,
  - afficher écarts de vitesse, fréquence cardiaque, puissance, cadence et segments communs,
  - indiquer si la sortie est atypique.
  Acceptance:
  - une activité donne immédiatement du contexte par rapport aux sorties comparables.

- [ ] `FUNC-P1-05` (`P1`, `M`) - Enrichir Routes avec difficulté et lisibilité terrain.
  Owners: `Product`, `Routes`, `Front`.
  Proposition:
  - afficher difficulté estimée, surface mix, part inconnue, confiance du profil OSRM et raisons de fallback directement sur la carte,
  - filtrer ou trier par `plus roulant`, `plus chemin`, `moins de demi-tours`, `plus familier`,
  - conserver les diagnostics techniques mais les traduire en signaux produit.
  Acceptance:
  - un utilisateur peut choisir une route sans lire les raisons brutes du moteur.

### Priorité basse

- [ ] `FUNC-P2-01` (`P2`, `M`) - Import local guidé GPX/FIT.
  Owners: `Product`, `Front`, `Back-Kotlin`, `Back-Go`.
  Proposition:
  - assistant de configuration pour sélectionner un dossier GPX/FIT,
  - validation de l'arborescence par année,
  - preview du nombre d'activités détectées et des champs manquants.
  Acceptance:
  - un usage sans Strava est compréhensible depuis l'UI.

- [ ] `FUNC-P2-02` (`P2`, `M`) - Calendrier d'entraînement unifié.
  Owners: `Product`, `Front`, `Stats`.
  Proposition:
  - vue calendrier combinant heatmap, charge hebdo, jours de repos, sorties longues et intensités,
  - navigation semaine/mois/année,
  - annotations manuelles locales.
  Acceptance:
  - lecture rapide de la régularité et des trous d'entraînement.

- [ ] `FUNC-P2-03` (`P2`, `S`) - Export enrichi.
  Owners: `Product`, `Front`, `Back-Go`, `Back-Kotlin`.
  Proposition:
  - exports CSV par vue avec les filtres appliqués,
  - export JSON des objectifs/routes sauvegardées,
  - export GPX groupé depuis la bibliothèque de routes.
  Acceptance:
  - les données importantes restent portables hors application.

## Dette visible à traiter en premier

- contrat OpenAPI partagé entre backends et frontend (`TECH-P1-01`),
- automatisation des checks routes encore manuels (`TECH-P1-02`),
- couverture frontend des parcours principaux (`TECH-P1-03`),
- fixtures de parité Go/Kotlin hors routes (`TECH-P1-05`).

## Vérification conseillée selon le type de changement

- Docs seulement: relecture Markdown.
- Front: `cd front-vue && npm run type-check && npm run test:unit`.
- Back Go: `cd back-go && go test ./...`.
- Back Kotlin: `cd back-kotlin && ./gradlew test`.
- Routes: lancer les tests ciblés Go/Kotlin + checks OSRM automatisés ou manuels documentés.
