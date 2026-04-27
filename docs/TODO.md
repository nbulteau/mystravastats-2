# TODO list

## Etat des lieux au 2026-04-25

- Monorepo avec trois surfaces principales: `front-vue`, `back-go`, `back-kotlin`.
- Le frontend Vue 3 couvre dashboard, objectifs annuels, diagnostics, charts, heatmap, statistiques, badges, activites, detail activite, segments, carte et routes.
- Les modes de source `STRAVA`, `FIT` et `GPX` existent, mais leur activation reste principalement une affaire de configuration runtime et de redemarrage backend.
- Le backend Go reste important pour le binaire local; le backend Kotlin reste la reference historique de plusieurs providers et services metier.
- La generation de routes reste la zone la plus sensible: OSRM, anti-retrace, diagnostics, export GPX, parite Go/Kotlin.
- Le prochain risque visible est la qualite des donnees locales FIT/GPX: valeurs invalides, streams incomplets, differences d'appareils et erreurs difficiles a expliquer depuis l'UI.
- La couverture frontend et la parite Go/Kotlin hors routes restent les meilleurs leviers pour eviter les regressions silencieuses.

## Garde-fous permanents

- Garder Go et Kotlin alignes pour tout changement de generation de routes.
- Ne jamais transformer l'historique en penalite de nouveaute: il doit rester un signal positif de corridors connus.
- Preserver les regles anti-retrace strictes hors zone depart/arrivee.
- Garder le comportement de zone depart/arrivee 2 km explicite et teste.
- Preserver `X-Request-Id` et les diagnostics exploitables sur les endpoints de generation.
- Ne pas changer silencieusement les contrats API: ajouter migration, compatibilite ou tests de contrat.
- Toute reponse JSON issue d'un provider local doit rester serialisable: pas de `NaN`, `Inf`, sentinelle FIT brute ou tableau `null` quand le contrat expose une liste.

## Chantiers techniques proposes

### Priorite haute

- [ ] `TECH-P0-05` (`P0`, `M`) - Smoke tests automatises des modes `STRAVA` / `FIT` / `GPX`.
  Owners: `QA`, `Back-Go`, `Back-Kotlin`, `Front`.
  Constat:
  - le mode source est transversal: runtime config, diagnostics, liste activites, detail activite, dashboard, cartes,
  - aujourd'hui on detecte encore certains problemes en cliquant dans l'UI.
  Scope:
  - creer un jeu minimal de fixtures locales FIT et GPX,
  - verifier pour chaque mode: `/api/health/details`, preview source, dashboard, liste activites, detail activite et maps GPX,
  - ajouter un script local unique qui lance le backend sur port temporaire et valide les endpoints critiques.
  Acceptance:
  - un changement provider se valide avec une commande reproductible,
  - les erreurs d'encodage detail activite sont detectees avant l'UI.

- [ ] `TECH-P1-01` (`P1`, `L`) - Mettre le contrat API sous controle OpenAPI partage.
  Owners: `Back-Kotlin`, `Back-Go`, `Front`, `QA`.
  Constat:
  - Springdoc existe cote Kotlin, Swagger existe cote Go, mais le frontend maintient ses interfaces a la main.
  Scope:
  - choisir une source de verite OpenAPI,
  - generer les types TypeScript et eventuellement un client API type,
  - ajouter des tests de conformite Go/Kotlin sur les DTO sensibles (`routes`, `statistics`, `dashboard`, `activities`, `source modes`, `annual goals`).
  Acceptance:
  - une divergence de champ ou d'enum casse la CI avant d'arriver dans l'UI.

- [ ] `TECH-P1-02` (`P1`, `M`) - Automatiser les checks routes aujourd'hui manuels.
  Owners: `QA`, `Back-Go`, `Back-Kotlin`, `Front`.
  Constat:
  - les docs de validation OSRM sont precises, mais plusieurs validations restent manuelles.
  Scope:
  - transformer les scenarios anti-retrace, direction, surface, fallback et shape tuning en smoke tests automatises,
  - lancer ces checks uniquement derriere profil CI/local OSRM pour eviter de ralentir la CI standard,
  - capturer les diagnostics cles en artifact.
  Acceptance:
  - un changement route peut etre valide avec une commande unique,
  - les cas terrain critiques restent reproductibles.

### Priorite moyenne

- [ ] `TECH-P1-03` (`P1`, `M`) - Etendre la couverture frontend.
  Owners: `Front`, `QA`.
  Constat:
  - les tests Vitest couvrent surtout stores/routes/charts utils,
  - les parcours UI riches sont peu proteges.
  Scope:
  - ajouter tests composants pour diagnostics, source mode, objectifs annuels, `HeaderBar`, `RoutesView`, `ActivityHeatmapChart`, `HeartRateZoneAnalysisPanel`,
  - ajouter quelques tests e2e/smoke avec backend mocke ou fixtures,
  - verifier les etats loading/erreur/cache et les erreurs API.
  Acceptance:
  - les workflows utilisateurs principaux sont proteges sans dependance a Strava.

- [ ] `TECH-P1-05` (`P1`, `L`) - Reduire le risque de divergence Go/Kotlin hors routes.
  Owners: `Back-Go`, `Back-Kotlin`, `QA`.
  Scope:
  - ajouter des fixtures partagees pour statistiques, badges, dashboard, heatmap, objectifs annuels, source modes et activites detaillees,
  - comparer au minimum les champs agreges et les cas limites de dates/streams manquants,
  - documenter les divergences acceptees quand une fonctionnalite n'existe que dans un backend.
  Acceptance:
  - la parite critique n'est plus limitee au moteur routes.

### Priorite basse

- [ ] `TECH-P2-01` (`P2`, `M`) - Nettoyer la strategie d'assets frontend embarques.
  Owners: `Front`, `Back-Kotlin`, `Back-Go`, `Infra`.
  Constat:
  - Kotlin contient des assets compiles dans `src/main/resources/static`,
  - Go embarque `public`,
  - le frontend a son propre build Vite.
  Scope:
  - definir si les assets compiles sont generes au build ou versionnes,
  - eviter les assets obsoletes dans les backends,
  - rendre les scripts de capture docs compatibles avec le mode retenu.
  Acceptance:
  - un build release ne peut pas embarquer une ancienne UI par accident.

- [ ] `TECH-P2-04` (`P2`, `L`) - Explorer un portage du backend local en Rust.
  Owners: `Back-Go`, `Rust`, `QA`, `Infra`.
  Proposition:
  - evaluer Rust comme cible possible pour le backend binaire local aujourd'hui porte par Go,
  - demarrer par un spike limite: health/details, lecture cache local, puis un provider local FIT ou GPX,
  - conserver strictement le contrat HTTP existant et comparer les reponses avec les fixtures Go/Kotlin,
  - mesurer taille du binaire, temps de demarrage, performance parsing FIT/GPX, complexite de packaging et ergonomie dev.
  Non-goal:
  - ne pas lancer une reecriture complete avant d'avoir des contrats OpenAPI et smoke tests source modes solides.
  Acceptance:
  - une note de decision compare Rust, Go et Kotlin sur les criteres projet,
  - un prototype expose quelques endpoints compatibles sans introduire de divergence fonctionnelle.

## Chantiers fonctionnels proposes

### Priorite haute

### Priorite moyenne

- [ ] `FUNC-P1-04` (`P1`, `M`) - Comparaison d'activite a effort similaire.
  Owners: `Product`, `Stats`, `Front`.
  Proposition:
  - dans le detail activite, comparer avec les sorties proches en distance/D+/sport/saison,
  - afficher ecarts de vitesse, frequence cardiaque, puissance, cadence et segments communs,
  - indiquer si la sortie est atypique.
  Acceptance:
  - une activite donne immediatement du contexte par rapport aux sorties comparables.

- [ ] `FUNC-P1-05` (`P1`, `M`) - Enrichir Routes avec difficulte et lisibilite terrain.
  Owners: `Product`, `Routes`, `Front`.
  Proposition:
  - afficher difficulte estimee, surface mix, part inconnue, confiance du profil OSRM et raisons de fallback directement sur la carte,
  - filtrer ou trier par `plus roulant`, `plus chemin`, `moins de demi-tours`, `plus familier`,
  - conserver les diagnostics techniques mais les traduire en signaux produit.
  Acceptance:
  - un utilisateur peut choisir une route sans lire les raisons brutes du moteur.

- [ ] `FUNC-P1-08` (`P1`, `M`) - Corrections locales non destructives des activites.
  Owners: `Product`, `Front`, `Back-Go`, `Back-Kotlin`.
  Proposition:
  - partir de l'audit qualite existant dans `Status`,
  - proposer des corrections locales non destructives: ignorer points GPS aberrants, recalculer distance/D+, masquer une valeur invalide,
  - conserver la donnee originale et un journal des corrections appliquees,
  - permettre dans la vue detail activite de basculer entre version brute et version corrigee,
  - afficher clairement les valeurs et traces modifiees quand la version corrigee est activee,
  - ajouter une action batch `Fix safe issues` pour appliquer uniquement les corrections non ambigues,
  - afficher un recapitulatif avant application: activites touchees, distance/D+ impactes, records potentiellement modifies,
  - conserver les anomalies risquees dans une revue manuelle plutot que de les corriger via un bouton global,
  - conserver la distinction deja posee entre stream complet absent, champ de stream structurel manquant et simple couverture capteur optionnelle,
  - permettre d'annuler une correction et d'expliquer son impact sur les statistiques.
  Acceptance:
  - une anomalie peut etre corrigee localement sans modifier la source STRAVA/FIT/GPX,
  - l'utilisateur voit clairement quelle valeur originale est remplacee ou ignoree,
  - la vue detail activite permet de comparer rapidement la sortie brute et la sortie corrigee,
  - les corrections batch sont limitees aux cas classes `safe` et restent annulables.

### Priorite basse

- [ ] `FUNC-P2-02` (`P2`, `M`) - Calendrier d'entrainement unifie.
  Owners: `Product`, `Front`, `Stats`.
  Proposition:
  - vue calendrier combinant heatmap, charge hebdo, jours de repos, sorties longues et intensites,
  - navigation semaine/mois/annee,
  - annotations manuelles locales.
  Acceptance:
  - lecture rapide de la regularite et des trous d'entrainement.

- [ ] `FUNC-P2-04` (`P2`, `S`) - Notes locales d'activite.
  Owners: `Product`, `Front`, `Back-Go`, `Back-Kotlin`.
  Proposition:
  - ajouter tags, notes libres et ressenti local sur une activite,
  - conserver ces donnees dans le cache local sans ecriture Strava,
  - afficher les notes dans detail activite, recherche et exports.
  Acceptance:
  - l'application peut enrichir les activites locales ou Strava sans modifier la source d'origine.

- [ ] `FUNC-P2-05` (`P2`, `S`) - Exports portables complementaires.
  Owners: `Product`, `Front`, `Back-Go`, `Back-Kotlin`.
  Proposition:
  - exporter en JSON les objectifs annuels et configurations locales,
  - exporter en GPX les traces locales filtrees,
  - documenter les schemas exportes pour pouvoir les reimporter plus tard.
  Acceptance:
  - les donnees de configuration et traces locales restent portables hors application.

## Dette visible a traiter en premier

- Smoke tests source modes (`TECH-P0-05`).
- Corrections locales non destructives des activites (`FUNC-P1-08`).
- Contrat OpenAPI partage (`TECH-P1-01`).

## Verification conseillee selon le type de changement

- Docs seulement: relecture Markdown.
- Front: `cd front-vue && npm run type-check && npm run test:unit`.
- Back Go: `cd back-go && go test ./...`.
- Back Kotlin: `cd back-kotlin && ./gradlew test`.
- Routes: lancer les tests cibles Go/Kotlin + checks OSRM automatises ou manuels documentes.
