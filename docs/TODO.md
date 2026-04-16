
### Améliorations techniques

#### Modularisation du backend Kotlin par fonctionnalité

**Contexte / problème :**
Le backend Kotlin, bien que structuré en couches (contrôleurs, services, adaptateurs), reste un module monolithique. À mesure que de nouvelles fonctionnalités sont ajoutées (badges, analyse de la fréquence cardiaque, etc.), la complexité du module principal augmente, ce qui peut rendre la maintenance et les tests plus difficiles.

**Proposition concrète :**
Diviser le projet `back-kotlin` en modules Gradle distincts, alignés sur les domaines fonctionnels :
- `core` : Modèles de domaine partagés, interfaces de service.
- `feature-statistics` : Implémentation des calculs de statistiques (best efforts, Eddington, etc.).
- `feature-badges` : Logique de calcul et d'attribution des badges.
- `feature-charts` : Préparation des données pour les graphiques.
- `infra-strava-adapter` : Communication avec l'API Strava et gestion du cache.
- `infra-storage` : Logique de persistance (que ce soit sur fichiers ou une base de données).
- `app` : Module principal qui assemble les fonctionnalités via l'injection de dépendances.

**Valeur attendue :**
- **Encapsulation :** Chaque fonctionnalité est isolée, ce qui facilite le raisonnement et les tests unitaires.
- **Scalabilité de l'équipe :** Plusieurs développeurs peuvent travailler sur des modules différents avec moins de risques de conflits.
- **Réutilisabilité :** Les modules de bas niveau (comme `core` ou `infra-storage`) pourraient être réutilisés plus facilement.

---

#### Contrat API typé et génération de clients partagés

**Contexte / problème :**
Le frontend Vue et les backends exposent de nombreuses routes `/api/...` avec des DTO qui évoluent dans le temps. Sans contrat unifié, les régressions de schéma (champ renommé, nullable inattendu, enum modifiée) sont détectées tardivement côté UI.

**Proposition concrète :**
Définir un contrat OpenAPI comme source de vérité (priorité backend Kotlin), puis générer automatiquement :
- les types TypeScript et un client API dans `front-vue`,
- des tests de conformité de contrat côté backend (snapshot OpenAPI + validations de sérialisation).
  Ajouter une vérification CI qui échoue si le code généré n'est pas à jour.

**Valeur attendue :**
Moins d'erreurs d'intégration frontend/backend, meilleure robustesse lors des refactorings de DTO, et onboarding simplifié grâce à une documentation API réellement exécutable.

---
#### Observabilité applicative et diagnostics guidés

**Contexte / problème :**
Les problèmes OAuth, rate limits Strava et incohérences de cache sont documentés, mais le diagnostic dépend encore fortement de l'inspection manuelle des logs et des fichiers.

**Proposition concrète :**
Ajouter un socle d'observabilité minimal :
- logs structurés avec `requestId` et catégories (`oauth`, `cache`, `strava`, `stats`),
- endpoint de santé enrichi (`/api/health/details`) indiquant état du cache, dernière synchro, et statut de quota Strava connu,
- page "Diagnostics" côté frontend exposant des checks lisibles pour l'utilisateur.

**Valeur attendue :**
Réduction du temps de support, débogage plus rapide en local/Docker, et meilleure confiance utilisateur lors des phases de première synchronisation.

---

### Améliorations fonctionnelles

#### Analyse de la charge d'entraînement (Training Load)

Actuellement, l'application calcule des métriques d'effort ponctuel (best efforts, records) mais ne propose pas de vision longitudinale de la charge d'entraînement cumulée.

**Proposition :**
Ajouter un indicateur de charge hebdomadaire et mensuelle inspiré du modèle CTL/ATL/TSB (Chronic Training Load / Acute Training Load / Training Stress Balance), calculable à partir des données disponibles : durée en mouvement, dénivelé, fréquence cardiaque (zones déjà calculées) et puissance (si disponible). Visualiser ces courbes dans l'onglet *Charts* pour permettre à l'athlète d'identifier des périodes de surcharge ou de sous-entraînement. Le backend dispose déjà des streams de fréquence cardiaque et des données de puissance, les ingrédients sont en place.

---

#### Objectifs annuels et projections de fin d'année

L'application affiche l'historique des performances mais ne permet pas à l'athlète de se fixer des objectifs et de visualiser sa progression vers ceux-ci.

**Proposition :**
Ajouter dans la vue *Dashboard* un bloc "Objectifs de l'année" où l'athlète définit des cibles (distance totale, dénivelé total, nombre d'Eddington cible, nombre de sorties). Pour chaque objectif, afficher :
- la progression actuelle (barre de progression + pourcentage),
- la date estimée d'atteinte basée sur la tendance des dernières semaines,
- un indicateur visuel (en avance / dans les temps / en retard) par rapport au rythme nécessaire.

Les objectifs seraient persistés dans le répertoire `strava-cache` (fichier JSON local par athlète), sans dépendance à Strava.

#### Plan d'entraînement adaptatif basé sur l'historique réel

**Contexte / problème :**
Les statistiques actuelles décrivent bien le passé, mais proposent peu d'aide prescriptive pour la suite (quoi faire cette semaine pour progresser sans surcharger).

**Proposition concrète :**
Créer un module "Plan adaptatif" qui suggère des volumes hebdomadaires par sport selon la tendance récente (charge, récupération, fréquence des sorties) et les objectifs choisis. Le module génère des recommandations simples : semaine allégée, maintien, ou progression.

**Valeur attendue :**
Passage d'une app descriptive à une app d'aide à la décision, avec un usage plus régulier entre deux sorties.

---

#### Explorateur d'itinéraires personnels et recommandations de sorties

**Contexte / problème :**
La carte affiche les activités, mais l'application n'exploite pas encore pleinement l'historique pour suggérer des parcours pertinents selon les préférences de l'athlète.

**Proposition concrète :**
Ajouter un explorateur "Sorties recommandées" qui propose :
- des boucles déjà réalisées proches d'une distance/dénivelé cible,
- des variantes "plus court / plus long / plus vallonné",
- des recommandations contextuelles par saison (profil similaire à vos meilleures sorties du printemps, etc.).
Basé uniquement sur les traces déjà présentes en cache (pas besoin d'API externe au départ).

**Valeur attendue :**
Expérience plus orientée usage terrain, meilleure réutilisation des données de carte, et différenciation fonctionnelle forte du produit.

---

#### Plan de delivery (Routes)

- [x] Sprint 1 - Stabilisation UX & contrat
  - 2 modes/contrats consolidés dans la couche Routes.
  - Harmonisation parsing/validation des query params Go/Kotlin.
  - Route Explorer intégré front/back avec filtres et cache court côté UI.

- [x] Sprint 2 - Qualité du moteur Target
  - Scoring amélioré sur distance + D+ + direction de départ.
  - Calibration des poids par type de parcours (`RIDE`, `MTB`, `GRAVEL`, `RUN`, `TRAIL`, `HIKE`).
  - Tests de non-régression Go/Kotlin pour garantir le ranking attendu.

- [x] Sprint 3 - Infra routage praticable (road-graph)
  - [x] Génération de nouvelles boucles praticables sur graphe routier (v1 beta sur graphe construit depuis le cache d'activités).
  - [x] Export GPX des routes générées (Go + Kotlin + Front).

- [ ] Sprint 4 - Shape-based generator avancé
  - Contrainte forme (dessin/import) avec projection sur routes praticables.
  - Variantes et scoring forme/km/D+.

---

#### Proposition détaillée - Routes v2 (Road-Graph + GPS Art)

Objectif produit:
- Passer d'un explorateur de sorties historiques à un vrai générateur de parcours praticables.
- Conserver une UX très simple: une seule carte, deux modes, export GPX immédiat.
- Produire des parcours réellement roulables/courables sur GPS (pas seulement "jolis" sur la carte).

Principes UX:
- Une carte unique sert à tout:
  - choisir le point de départ,
  - dessiner/importer une forme en mode shape,
  - afficher et comparer les routes générées.
- Deux modes explicites:
  - `Target loop generator`
  - `Shape based generator`
- CTA principal clair:
  - `Generate loop` en mode target,
  - `Generate from shape` en mode shape.
- `Use my location` visible et prioritaire.
- Export GPX présent à côté de la carte pour chaque variante.

### Mode 1 - Target loop generator (priorité utilitaire)

Entrées:
- Type de parcours: Ride, VTT, Gravel, Course à pied, Trail, Randonnée.
- Direction de départ: Nord, Sud, Est, Ouest.
- Distance cible (km).
- Dénivelé cible (m).
- Point de départ via carte (pas de saisie lat/lng manuelle).

Sorties:
- 1 meilleure boucle + variantes (ex: plus courte, plus longue, plus vallonnée).
- Score explicite par contrainte:
  - respect distance,
  - respect D+,
  - respect direction de départ,
  - praticabilité route réseau.
- Export GPX direct.

Moteur:
- Routage sur graphe routier (OSM via moteur local type OSRM/GraphHopper/Valhalla).
- Coûts par type de pratique:
  - vélo route: pénaliser segments non revêtus/risqués,
  - gravel/VTT/trail: tolérer davantage chemins adaptés.
- Fonction objectif:
  - `score = w1*erreur_km + w2*erreur_D+ + w3*erreur_direction + w4*coût_praticabilité`.

### Mode 2 - Shape based generator (GPS Art)

Entrées:
- Forme fournie par:
  - dessin libre sur la carte,
  - import GPX,
  - import SVG/polyline.
- Échelle distance cible (km).
- Point de départ (optionnel si on veut imposer un ancrage).
- Type de parcours (contraintes de praticabilité du profil).

Pipeline shape-to-route:
1. Normaliser la forme:
   - simplification (Douglas-Peucker),
   - rééchantillonnage régulier,
   - normalisation translation/rotation/échelle.
2. Placement géographique:
   - ancrage proche du point de départ choisi,
   - recherche locale de placement minimisant les segments impossibles.
3. Projection sur réseau routier:
   - snap des points-clés vers le graphe,
   - routage entre points successifs.
4. Optimisation multi-objectif:
   - minimiser distance à la forme,
   - minimiser erreur km cible,
   - minimiser erreur D+ cible,
   - minimiser non-praticabilité.
5. Génération de variantes:
   - stricte forme,
   - équilibrée,
   - plus roulante.

Sorties:
- 3 à 5 variantes scorées.
- Indicateurs lisibles:
  - shape similarity (%),
  - erreur distance,
  - erreur D+,
  - confiance praticabilité.
- Export GPX immédiat pour chaque variante.

### Inspirations produit observées

Ce qui est pertinent à reprendre:
- `gpsartify`:
  - focus "generator" simple orienté résultat,
  - custom shape + téléchargement de routes.
- `routedoodle`:
  - modèle en couches `Picture -> Wireframe -> Route`,
  - conversion automatique de points de contrôle en waypoints routés,
  - modes de routage par pratique (bike/pedestrian),
  - export GPX/TCX et workflow GPS art pragmatique.

Ce qu'on garde différent pour MyStravaStats:
- priorité à la praticabilité réelle et à l'usage GPS sportif,
- mode target utilitaire au même niveau que le mode art,
- intégration forte au cache historique et aux profils sportifs déjà connus.

### Contrat API cible (Go + Kotlin identique)

Endpoints:
- `POST /api/routes/generate/target`
- `POST /api/routes/generate/shape`
- `GET /api/routes/{routeId}/gpx`

Payload target:
- `startPoint {lat,lng}`
- `routeType`
- `startDirection`
- `distanceTargetKm`
- `elevationTargetM`
- `variantCount`

Payload shape:
- `shapeInputType` (`draw`, `gpx`, `svg`, `polyline`)
- `shapeData`
- `startPoint` (optionnel)
- `distanceTargetKm`
- `elevationTargetM` (optionnel)
- `routeType`
- `variantCount`

Réponse commune:
- `routes[]` avec:
  - géométrie preview,
  - métriques (km, D+, durée estimée),
  - score global + sous-scores,
  - drapeau `isRoadGraphGenerated`,
  - `routeId`.

### Persistance/cache

- Persister les routes générées (manifest cache unifié déjà en place):
  - éviter de recalculer une requête identique,
  - permettre ré-ouverture rapide de la dernière génération.
- Indexer par clé de requête normalisée:
  - mode + profil + contraintes + hash de forme.
- TTL configurable + invalidation explicite via bouton `Refresh`.

### Plan de livraison recommandé

Sprint A - Contrat + UX carte unique:
- écran routes consolidé en 2 modes explicites,
- picker start point + use my location + export GPX latéral,
- API `generate/target` + `generate/shape` (stub stable Go/Kotlin).

Sprint B - Target engine robuste:
- génération boucle road-graph fiable,
- calibration scoring par type de pratique,
- tests de non-régression sur km/D+/direction.

Sprint C - Shape engine v1:
- import polyline/GPX,
- normalisation de forme + projection routière,
- variantes scorées + export GPX.

Sprint D - Shape engine avancé:
- import SVG + dessin libre assisté,
- optimisation plus fine (forme vs praticabilité),
- instrumentation qualité (taux de réussite, temps de calcul, satisfaction score).

### Definition of Done (v2)

- Les deux modes sont utilisables en production sur la même carte.
- Chaque génération renvoie au moins une route praticable ou un message explicite de blocage.
- Export GPX fonctionne pour chaque route proposée.
- Le score explique clairement les compromis (forme, km, D+, praticabilité).
- Parité fonctionnelle Go/Kotlin validée par tests de contrat.

### Backlog exécutable (tickets prêts à coder)

Convention:
- Priorité: `P0` (bloquant MVP), `P1` (important), `P2` (amélioration).
- Taille: `S` (<= 1 jour), `M` (1-3 jours), `L` (3-5 jours), `XL` (> 1 sprint).
- Owners possibles: `Front`, `Back-Go`, `Back-Kotlin`, `Infra`, `QA`.

#### Sprint A - Contrat + UX carte unique

- [x] `ROUTE-A01` (`P0`, `M`) - Unifier l'écran Routes en 2 modes explicites.
  Owners: `Front`.
  Dépendances: aucune.
  Scope: switch `Target loop generator` / `Shape based generator`, champs conditionnels par mode, suppression des champs non nécessaires.
  Acceptance: le mode actif est persisté dans le store; aucun champ hors mode n'est affiché; pas de régression sur l'export GPX existant.

- [x] `ROUTE-A02` (`P0`, `M`) - Carte unique shared state.
  Owners: `Front`.
  Dépendances: `ROUTE-A01`.
  Scope: une seule carte pour start picker, dessin/import shape, preview résultats.
  Acceptance: changement de mode sans remonter/recréer la carte; start point et shape restent synchronisés avec le store.

- [x] `ROUTE-A03` (`P0`, `S`) - Bouton `Use my location` + fallback propre.
  Owners: `Front`.
  Dépendances: `ROUTE-A02`.
  Scope: géolocalisation navigateur avec feedback utilisateur (toast/info erreur permissions).
  Acceptance: clique => centre + marqueur start; en refus permission message clair et UI non bloquée.

- [x] `ROUTE-A04` (`P0`, `M`) - Contrat API v2 Target (Go).
  Owners: `Back-Go`.
  Dépendances: aucune.
  Scope: endpoint `POST /api/routes/generate/target`, validation payload, réponse standardisée `routes[]` + scores.
  Acceptance: tests handlers + contrat JSON; erreurs 400 homogènes.

- [x] `ROUTE-A05` (`P0`, `M`) - Contrat API v2 Target (Kotlin).
  Owners: `Back-Kotlin`.
  Dépendances: aucune.
  Scope: endpoint équivalent Kotlin avec même schéma de réponse/erreur que Go.
  Acceptance: tests controller + sérialisation DTO alignée avec Go.

- [x] `ROUTE-A06` (`P0`, `M`) - Contrat API v2 Shape (Go + Kotlin).
  Owners: `Back-Go`, `Back-Kotlin`.
  Dépendances: `ROUTE-A04`, `ROUTE-A05`.
  Scope: endpoint `POST /api/routes/generate/shape` acceptant `draw/gpx/svg/polyline` (au moins `draw/polyline` dans ce sprint).
  Acceptance: même schéma de réponse sur les deux backends; validation de `shapeInputType`.

- [x] `ROUTE-A07` (`P1`, `S`) - Export GPX latéral immédiat.
  Owners: `Front`.
  Dépendances: `ROUTE-A04`, `ROUTE-A05`.
  Scope: panneau de résultats avec bouton export par route, état "exporting", gestion erreur.
  Acceptance: téléchargement GPX fonctionne pour chaque variante affichée.

- [ ] `ROUTE-A08` (`P1`, `S`) - Telemetry de base routes.
  Owners: `Front`, `Back-Go`, `Back-Kotlin`.
  Dépendances: `ROUTE-A04`, `ROUTE-A05`.
  Scope: logs structurés `route_mode`, `generation_ms`, `variant_count`, `export_success`.
  Acceptance: traces visibles en logs backend + console dev front.

#### Sprint B - Target engine robuste (road-graph utile terrain)

- [ ] `ROUTE-B01` (`P0`, `L`) - Intégration moteur routage local.
  Owners: `Infra`, `Back-Go`, `Back-Kotlin`.
  Dépendances: `ROUTE-A04`, `ROUTE-A05`.
  Scope: abstraction `RoutingEnginePort` + impl locale (OSRM/GraphHopper), config Docker/dev.
  Acceptance: healthcheck routage; appel simple entre 2 points OK.

- [ ] `ROUTE-B02` (`P0`, `L`) - Génération boucle target (Go).
  Owners: `Back-Go`.
  Dépendances: `ROUTE-B01`.
  Scope: algo boucle contraint par start point, direction initiale, distance cible, D+ cible.
  Acceptance: au moins 1 route praticable pour cas standard; timeout contrôlé.

- [ ] `ROUTE-B03` (`P0`, `L`) - Génération boucle target (Kotlin).
  Owners: `Back-Kotlin`.
  Dépendances: `ROUTE-B01`.
  Scope: même logique/contrat que Go.
  Acceptance: parité de réponse fonctionnelle avec Go sur dataset de test commun.

- [ ] `ROUTE-B04` (`P1`, `M`) - Calibration par type de pratique.
  Owners: `Back-Go`, `Back-Kotlin`.
  Dépendances: `ROUTE-B02`, `ROUTE-B03`.
  Scope: profils de coût route/gravel/mtb/run/trail/hike.
  Acceptance: tests non-régression montrant des rankings différents par profil.

- [ ] `ROUTE-B05` (`P1`, `M`) - Score explainability target.
  Owners: `Back-Go`, `Back-Kotlin`, `Front`.
  Dépendances: `ROUTE-B02`, `ROUTE-B03`.
  Scope: sous-scores `distance`, `elevation`, `direction`, `roadFitness` dans la réponse + rendu UI.
  Acceptance: UI affiche les sous-scores et la raison de sélection.

- [ ] `ROUTE-B06` (`P1`, `M`) - Tests non-régression target.
  Owners: `QA`, `Back-Go`, `Back-Kotlin`.
  Dépendances: `ROUTE-B02`, `ROUTE-B03`, `ROUTE-B04`.
  Scope: fixtures synthétiques + snapshots de classement.
  Acceptance: tests GIVEN/WHEN/THEN sur cas limites (impossible D+, distance courte, direction inversée).

#### Sprint C - Shape engine v1 (GPS Art exploitable)

- [ ] `ROUTE-C01` (`P0`, `M`) - Import polyline/GPX côté front.
  Owners: `Front`.
  Dépendances: `ROUTE-A02`, `ROUTE-A06`.
  Scope: upload/parse GPX, collage polyline, visualisation wireframe.
  Acceptance: forme importée visible sur carte et stockée dans le store.

- [ ] `ROUTE-C02` (`P0`, `M`) - Normalisation de forme backend.
  Owners: `Back-Go`, `Back-Kotlin`.
  Dépendances: `ROUTE-A06`.
  Scope: simplification + rééchantillonnage + normalisation rotation/scale.
  Acceptance: shape canonique déterministe pour une même entrée.

- [ ] `ROUTE-C03` (`P0`, `L`) - Projection shape -> road graph.
  Owners: `Back-Go`, `Back-Kotlin`.
  Dépendances: `ROUTE-B01`, `ROUTE-C02`.
  Scope: snap des points de contrôle + routage inter-points.
  Acceptance: au moins une route praticable générée pour shape simple.

- [ ] `ROUTE-C04` (`P1`, `M`) - Scoring multi-objectif shape.
  Owners: `Back-Go`, `Back-Kotlin`.
  Dépendances: `ROUTE-C03`.
  Scope: score forme + erreur km + erreur D+ + praticabilité.
  Acceptance: score cohérent affiché et trié dans le résultat.

- [ ] `ROUTE-C05` (`P1`, `M`) - Variantes shape (3 profils).
  Owners: `Back-Go`, `Back-Kotlin`, `Front`.
  Dépendances: `ROUTE-C04`.
  Scope: `strict shape`, `balanced`, `road-friendly`.
  Acceptance: 3 variantes avec trade-offs lisibles et exportables.

- [ ] `ROUTE-C06` (`P1`, `S`) - Empty/error UX shape.
  Owners: `Front`.
  Dépendances: `ROUTE-C03`.
  Scope: messages explicites "shape impossible localement", conseils de retry (scale, point de départ).
  Acceptance: pas de page vide ni erreur silencieuse.

#### Sprint D - Shape engine avancé + qualité

- [ ] `ROUTE-D01` (`P1`, `M`) - Import SVG (paths simples).
  Owners: `Front`, `Back-Go`, `Back-Kotlin`.
  Dépendances: `ROUTE-C02`.
  Scope: parser SVG -> polyline normalisée.
  Acceptance: un SVG simple produit une génération shape valide.

- [ ] `ROUTE-D02` (`P1`, `L`) - Dessin libre assisté (éditeur wireframe).
  Owners: `Front`.
  Dépendances: `ROUTE-A02`.
  Scope: move/scale/rotate shape directement sur carte.
  Acceptance: manipulation fluide, undo/clear minimum.

- [ ] `ROUTE-D03` (`P1`, `M`) - Cache des générations routes v2.
  Owners: `Back-Go`, `Back-Kotlin`.
  Dépendances: `ROUTE-B02`, `ROUTE-C03`.
  Scope: persistance par clé normalisée (mode+profil+contraintes+hash shape) dans le manifest cache.
  Acceptance: requête identique servie depuis cache avec hit observable.

- [ ] `ROUTE-D04` (`P2`, `S`) - Bench & guardrails performance.
  Owners: `QA`, `Back-Go`, `Back-Kotlin`.
  Dépendances: `ROUTE-D03`.
  Scope: budget de latence, timeout, circuit-breaker routage.
  Acceptance: p95 mesurée; dégradation propre en cas de moteur indisponible.

- [ ] `ROUTE-D05` (`P2`, `S`) - Contrat de parité automatique Go/Kotlin.
  Owners: `QA`, `Back-Go`, `Back-Kotlin`.
  Dépendances: `ROUTE-B03`, `ROUTE-C05`.
  Scope: tests de contrat croisés sur fixtures communes.
  Acceptance: CI rouge si divergence schéma ou comportement critique.
