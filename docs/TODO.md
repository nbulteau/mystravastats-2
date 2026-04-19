
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

#### Routes (OSRM) - backlog restant uniquement

Objectif produit conservé:
- générer des boucles praticables depuis un point de départ,
- 2 modes (`Target loop generator`, `Shape based generator`),
- export GPX immédiat,
- parité Go/Kotlin.

Ce qui est déjà fait (retiré du backlog):
- contrat API target/shape unifié,
- carte unique et UX principale (`Use my location`, `Generate route`, export GPX),
- intégration OSRM + endpoint health routing,
- base de scoring distance/D+/direction,
- fallback de route type (`MTB -> GRAVEL -> RIDE`).

### Priorités restantes

- [ ] `ROUTE-P0-01` (`P0`, `L`) - Stabiliser la génération target pour ne plus revenir à `0 route`.
  Owners: `Back-Go`, `Back-Kotlin`.
  Scope:
  - pipeline d'assouplissement déterministe (strict -> relax -> best-effort) identique sur Go/Kotlin,
  - garantie: si une boucle `Ride` valide existe, ne jamais renvoyer "no candidate" en `Gravel`/`MTB`,
  - diagnostics normalisés et non contradictoires (`NO_CANDIDATE`, `BACKTRACKING_FILTERED`, etc.).
  Acceptance:
  - tests de non-régression verts,
  - cas réel: `40km / 800m` génère une route sur zone urbaine dense.

- [ ] `ROUTE-P0-02` (`P0`, `L`) - Anti-retours robuste hors zone départ/arrivée (2 km).
  Owners: `Back-Go`, `Back-Kotlin`.
  Scope:
  - contrainte dure: pas de réutilisation d'axe OSM hors zone 2 km (même sens ou sens inverse),
  - autoriser uniquement les croisements géométriques et la zone de retour,
  - pénalités fortes sur corridor overlap + edge reuse dans le ranking final.
  Acceptance:
  - baisse nette des aller/retour sur les GPX générés,
  - tests dédiés sur la métrique de réutilisation d'axes.

- [ ] `ROUTE-P0-03` (`P0`, `M`) - Direction "globale" non bloquante.
  Owners: `Back-Go`, `Back-Kotlin`.
  Scope:
  - `Direction` influence l'orientation moyenne de la boucle,
  - ne doit plus bloquer la génération,
  - fallback automatique vers direction relâchée avec raison explicite.
  Acceptance:
  - génération réussie avec et sans direction,
  - la boucle respecte majoritairement le quadrant demandé quand possible.

- [ ] `ROUTE-P1-01` (`P1`, `L`) - Vrai scoring surface (OSM tags `surface` / `tracktype`).
  Owners: `Back-Go`, `Back-Kotlin`, `Infra`.
  Scope:
  - enrichir les segments routés pour récupérer la typologie de revêtement,
  - appliquer les règles métier:
    - `Ride`: privilégier asphalt/paved,
    - `Gravel`: minimum 25% chemins (fallback Ride si impossible),
    - `MTB`: maximiser chemins/track.
  Acceptance:
  - différence visible de parcours entre `Ride`, `Gravel`, `MTB`,
  - tests de classement par type de surface.

- [ ] `ROUTE-P1-02` (`P1`, `M`) - Statut moteur OSRM + profil réellement exploité dans l'UI.
  Owners: `Front`, `Back-Go`, `Back-Kotlin`.
  Scope:
  - afficher état `online/offline/misconfigured`,
  - afficher profil actif (`bicycle.lua`, `foot.lua`, `car.lua`) et route types effectivement autorisés,
  - désactiver côté UI les types incompatibles avec le profil.
  Acceptance:
  - cohérence entre health backend et options UI,
  - plus d'option sélectionnable invalide.

- [ ] `ROUTE-P1-03` (`P1`, `M`) - Génération incrémentale côté UI (1 clic = 1 nouvelle route unique).
  Owners: `Front`, `Back-Go`, `Back-Kotlin`.
  Scope:
  - à chaque clic `Generate route`, ajouter une route nouvelle,
  - déduplication géométrique stricte,
  - conserver l'historique de session et sélection active.
  Acceptance:
  - plus de doublons dans `Generated routes`,
  - UX claire "last generated".

- [ ] `ROUTE-P1-04` (`P1`, `L`) - Shape mode v1 utilisable terrain.
  Owners: `Front`, `Back-Go`, `Back-Kotlin`.
  Scope:
  - import polyline/GPX stable,
  - projection shape -> réseau routier,
  - au moins 2 variantes scorées (shape-first / road-first),
  - export GPX par variante.
  Acceptance:
  - une forme simple produit au moins une route praticable.

- [ ] `ROUTE-P2-01` (`P2`, `M`) - Observabilité routes.
  Owners: `Back-Go`, `Back-Kotlin`, `Front`.
  Scope:
  - logs structurés (`requestId`, mode, routeType, target km/D+, temps de génération, raisons rejet),
  - résumé compact affichable côté UI en cas d'échec.
  Acceptance:
  - diagnostic actionnable en moins d'1 minute sans lire tout le log brut.

- [ ] `ROUTE-P2-02` (`P2`, `M`) - Parité automatique Go/Kotlin.
  Owners: `QA`, `Back-Go`, `Back-Kotlin`.
  Scope:
  - fixtures communes + assertions de contrat sur le top résultat,
  - comparaison des diagnostics de rejet.
  Acceptance:
  - CI rouge si divergence de comportement critique.

### Definition of Done (mise à jour)

- `Target` génère une boucle praticable dans >90% des cas de test locaux.
- Hors zone 2 km autour du départ/arrivée, pas de réutilisation d'axe (même sens ou inverse).
- `Gravel` et `MTB` diffèrent réellement de `Ride` sur la part de chemins.
- Plus de "0 route" tant qu'une solution `Ride` valide existe.
- Parité Go/Kotlin validée en CI sur fixtures partagées.
