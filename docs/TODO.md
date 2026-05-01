# TODO list

## Etat des lieux au 2026-04-28

- Monorepo avec trois surfaces principales: `front-vue`, `back-go`, `back-kotlin`.
- Le frontend Vue 3 couvre dashboard, objectifs annuels, diagnostics, source modes, data quality, charts, heatmap, statistiques, badges, activites, detail activite, segments, carte, materiel et routes.
- Les modes de source `STRAVA`, `FIT` et `GPX` existent dans Go et Kotlin. Leur activation reste principalement une affaire de configuration runtime et de redemarrage backend.
- Le backend Go reste important pour le binaire local; le backend Kotlin reste la reference historique de plusieurs providers et services metier.
- La generation de routes reste la zone la plus sensible: OSRM, anti-retrace, diagnostics, export GPX, parite Go/Kotlin.
- L'onglet routes est en cours de repositionnement en `Strava Art` / GPS drawing studio: dessiner ou importer une forme, la snapper au reseau routier via OSRM, puis exporter un GPX exploitable.
- La qualite des donnees locales FIT/GPX a deja un socle de diagnostics et corrections locales. Le risque suivant est la validation reproductible: fixtures, smoke tests et comparaison avant/apres correction.
- La couverture frontend, le contrat API partage et la parite Go/Kotlin hors routes restent les meilleurs leviers pour eviter les regressions silencieuses.

## Garde-fous permanents

- Garder Go et Kotlin alignes pour tout changement de generation de routes.
- Ne jamais transformer l'historique en penalite de nouveaute: il doit rester un signal positif de corridors connus.
- Preserver les regles anti-retrace strictes hors zone depart/arrivee.
- Garder le comportement de zone depart/arrivee 2 km explicite et teste.
- Preserver `X-Request-Id` et les diagnostics exploitables sur les endpoints de generation.
- Pour `Strava Art`, conserver `/routes` comme URL interne tant qu'aucune migration n'est prevue.
- Pour `Strava Art`, rendre visibles le dessin d'origine, la route OSRM generee, les scores de ressemblance/praticabilite et les raisons de fallback.
- Pour `Strava Art`, le score `Art fit` doit rester centre sur le respect du dessin: proximite ancree, derive du centre, ordre du trace et forme globale.
- Garder les exports GPX generes compatibles avec Strava, Garmin, Komoot et les outils GPS standards.
- Ne pas changer silencieusement les contrats API: ajouter migration, compatibilite ou tests de contrat.
- Toute reponse JSON issue d'un provider local doit rester serialisable: pas de `NaN`, `Inf`, sentinelle FIT brute ou tableau `null` quand le contrat expose une liste.
- Toute correction locale doit rester reversible et explicite dans les diagnostics.

## Chantiers techniques proposes

### Priorite haute

- [ ] `TECH-P0-05` (`P0`, `M`) - Smoke tests automatises des modes `STRAVA` / `FIT` / `GPX`.
  Owners: `QA`, `Back-Go`, `Back-Kotlin`, `Front`.
  Constat:
  - le mode source est transversal: runtime config, diagnostics, liste activites, detail activite, dashboard, cartes et data quality,
  - les tests unitaires providers existent, mais les ruptures de parcours complet se detectent encore trop tard.
  Scope:
  - creer un jeu minimal de fixtures locales FIT et GPX dans `test-fixtures`,
  - verifier pour chaque mode: `/api/health/details`, preview source, dashboard, liste activites, detail activite, maps GPX et rapport data quality,
  - ajouter un script local unique qui lance le backend sur port temporaire et valide les endpoints critiques,
  - verifier explicitement que les reponses JSON restent serialisables.
  Acceptance:
  - un changement provider se valide avec une commande reproductible,
  - les erreurs d'encodage detail activite ou data quality sont detectees avant l'UI.

- [ ] `TECH-P1-01` (`P1`, `L`) - Mettre le contrat API sous controle OpenAPI partage.
  Owners: `Back-Kotlin`, `Back-Go`, `Front`, `QA`.
  Constat:
  - Springdoc existe cote Kotlin, Swagger existe cote Go, mais le frontend maintient ses interfaces a la main,
  - plusieurs DTO sensibles evoluent dans deux backends.
  Scope:
  - choisir une source de verite OpenAPI ou un mode de comparaison strict entre specs,
  - generer les types TypeScript et eventuellement un client API type,
  - ajouter des tests de conformite Go/Kotlin sur les DTO sensibles (`routes`, `statistics`, `dashboard`, `activities`, `source modes`, `data quality`, `gear analysis`, `annual goals`).
  Acceptance:
  - une divergence de champ ou d'enum casse la CI avant d'arriver dans l'UI.

- [ ] `TECH-P1-02` (`P1`, `M`) - Automatiser les checks routes aujourd'hui manuels.
  Owners: `QA`, `Back-Go`, `Back-Kotlin`, `Front`.
  Constat:
  - les docs de validation OSRM sont precises,
  - les scripts `manual-route-*` restent dependants d'un lancement local et d'une interpretation humaine.
  Scope:
  - transformer les scenarios anti-retrace, direction, surface, fallback et shape tuning en smoke tests automatises,
  - lancer ces checks uniquement derriere profil CI/local OSRM pour eviter de ralentir la CI standard,
  - capturer les diagnostics cles en artifact.
  Acceptance:
  - un changement route peut etre valide avec une commande unique,
  - les cas terrain critiques restent reproductibles.

- [ ] `TECH-P1-06` (`P1`, `M`) - Stabiliser les fixtures de qualite de donnees locales.
  Owners: `QA`, `Back-Go`, `Back-Kotlin`.
  Constat:
  - les diagnostics et corrections data quality existent,
  - il manque un corpus partage qui couvre valeurs invalides, streams incomplets, GPS aberrant, altitude spike et fichiers locaux limites.
  Scope:
  - ajouter fixtures FIT/GPX anonymisees et petites,
  - comparer les rapports data quality Go/Kotlin sur les categories, severites, champs et corrections proposees,
  - ajouter un snapshot lisible des impacts avant/apres correction.
  Acceptance:
  - une evolution de parsing local ou de correction casse un test quand elle change le diagnostic attendu.

### Priorite moyenne

- [ ] `TECH-P1-03` (`P1`, `M`) - Etendre la couverture frontend.
  Owners: `Front`, `QA`.
  Constat:
  - les tests Vitest couvrent surtout stores/routes/charts utils,
  - les parcours UI riches sont peu proteges.
  Scope:
  - ajouter tests composants pour diagnostics, source mode, data quality, objectifs annuels, `HeaderBar`, `RoutesView`, `ActivityHeatmapChart`, `HeartRateZoneAnalysisPanel` et `GearAnalysisView`,
  - ajouter quelques tests e2e/smoke avec backend mocke ou fixtures,
  - verifier les etats loading/erreur/cache et les erreurs API.
  Acceptance:
  - les workflows utilisateurs principaux sont proteges sans dependance a Strava.

- [ ] `TECH-P1-05` (`P1`, `L`) - Reduire le risque de divergence Go/Kotlin hors routes.
  Owners: `Back-Go`, `Back-Kotlin`, `QA`.
  Scope:
  - ajouter des fixtures partagees pour statistiques, badges, dashboard, heatmap, objectifs annuels, source modes, data quality, gear analysis et activites detaillees,
  - comparer au minimum les champs agreges et les cas limites de dates/streams manquants,
  - documenter les divergences acceptees quand une fonctionnalite n'existe que dans un backend.
  Acceptance:
  - la parite critique n'est plus limitee au moteur routes.

- [ ] `TECH-P1-07` (`P1`, `S`) - Eviter la derive des docs de capacites backend.
  Owners: `Docs`, `Back-Go`, `Back-Kotlin`.
  Constat:
  - les capacites providers evoluent plus vite que certaines pages d'architecture,
  - la matrice backend, runtime config et docs source modes doivent rester coherentes.
  Scope:
  - ajouter une checklist de mise a jour docs quand `/api/health/details` ou les providers changent,
  - verifier que la matrice de capacites reflete les tests runtime,
  - faire pointer les docs source modes vers les memes regles de selection `FIT_FILES_PATH`, `GPX_FILES_PATH`, `STRAVA_CACHE_PATH`.
  Acceptance:
  - un changement de capacite backend ne laisse plus une doc contradictoire.

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

- [ ] `TECH-P2-05` (`P2`, `M`) - Clarifier la strategie long terme des backends.
  Owners: `Back-Go`, `Back-Kotlin`, `Product`, `QA`.
  Proposition:
  - decider explicitement quelles responsabilites restent doubles, quelles surfaces deviennent reference Go, et quelles surfaces restent reference Kotlin,
  - eviter les reecritures exploratoires tant que les contrats et fixtures de parite ne sont pas solides,
  - documenter les criteres de choix: distribution locale, performance parsing FIT/GPX, maturite providers, cout de maintenance et ergonomie dev.
  Acceptance:
  - une note de decision remplace les idees de portage premature par une strategie testable.

## Chantiers fonctionnels proposes

### Priorite haute

- [ ] `FUNC-P0-01` (`P0`, `M`) - Parcours d'import local guide.
  Owners: `Product`, `Front`, `Back-Go`, `Back-Kotlin`.
  Proposition:
  - transformer la preview source mode en parcours complet: verifier dossier, expliquer champs manquants, proposer commande de lancement, puis verifier le mode actif apres redemarrage,
  - afficher un recap utilisable avant de quitter le mode courant: activites trouvees, annees, erreurs bloquantes, qualite des donnees et prochaines actions,
  - rendre les differences STRAVA/FIT/GPX comprensibles sans lire la config runtime.
  Acceptance:
  - un utilisateur peut passer d'une source Strava a un dossier FIT/GPX local sans deviner les variables d'environnement.

- [ ] `FUNC-P0-02` (`P0`, `M`) - Triage data quality centre utilisateur.
  Owners: `Product`, `Front`, `Stats`.
  Proposition:
  - ajouter une file de revue des problemes data quality avec filtre par severite, activite, champ et impact,
  - afficher les differences avant/apres correction sur distance, D+, vitesse max et records potentiellement impactes,
  - grouper les actions sures, manuelles et non supportees.
  Acceptance:
  - un utilisateur sait quelles corrections appliquer, lesquelles ignorer et lesquelles peuvent changer ses statistiques.

- [ ] `FUNC-P0-03` (`P0`, `XL`, `EPIC`) - GPS drawing studio / Strava Art.
  Owners: `Product`, `Routes`, `Front`, `Back-Go`, `Back-Kotlin`, `QA`.
  Objectif:
  - transformer l'onglet `Routes` en atelier de creation GPS art: partir d'un dessin, d'une forme ou d'un GPX existant, produire une route praticable qui ressemble au dessin, puis exporter un GPX.
  Positionnement produit:
  - onglet visible `Strava Art`, URL interne `/routes` conservee,
  - mode unique `Draw art`,
  - plus de mode `Generate loop`,
  - plus de cible sportive `Distance target` ou `Elevation target`,
  - le dessin/la forme est la contrainte principale; la distance et le denivele sont des resultats affiches, pas des objectifs saisis.
  Inspirations:
  - RouteSketcher: dessin libre, snapping au reseau routier, import GPX editable, deplacement/rotation/scale du croquis,
  - GPSArtify: generation orientee Strava Art, route proche de la position utilisateur, workflow simple planifier/enregistrer/partager,
  - gps2gpx.art: conversion dessin vers GPX avec experience courte et export direct.
  Decoupage:
  - `FUNC-P0-03A` - Contrat API Strava Art et nettoyage legacy,
  - `FUNC-P0-03B` - Smoke tests generation + export GPX,
  - `FUNC-P0-03C` - UX MVP dessin/import/generation,
  - `FUNC-P0-03D` - Resultats lisibles et diagnostics produit,
  - `FUNC-P0-03E` - Comparaison visuelle dessin original vs route generee,
  - `FUNC-P0-03F` - Outils de transformation du dessin,
  - `FUNC-P0-03G` - Bibliotheque de formes et sauvegarde locale,
  - `FUNC-P0-03H` - Exports avances et assistant de correction.
  Acceptance epic:
  - un utilisateur peut dessiner ou importer un GPX, lancer la generation, choisir une proposition et exporter un GPX,
  - chaque proposition indique clairement si l'art est lisible et praticable,
  - le GPX exporte est exploitable dans une application GPS standard,
  - un echec de generation explique les contraintes bloqueantes ou les fallbacks.

- [x] `FUNC-P0-03A` (`P0`, `M`) - Contrat API Strava Art et nettoyage legacy.
  Owners: `Routes`, `Back-Go`, `Back-Kotlin`, `Front`.
  Scope:
  - onglet visible `Strava Art`, URL interne `/routes` conservee,
  - mode unique `Draw art`,
  - supprimer le mode public `Generate loop`,
  - supprimer les objectifs utilisateur `Distance target` et `Elevation target`,
  - endpoint public unique de generation: `POST /api/routes/generate/shape`,
  - export: `GET /api/routes/{routeId}/gpx`,
  - retirer du payload Strava Art: `distanceTargetKm`, `elevationTargetM`, `startDirection`, `generationMode`, `customWaypoints`.
  Acceptance:
  - Go et Kotlin exposent le meme contrat public,
  - le store Vue n'appelle plus `/api/routes/generate/target`,
  - les DTO de route generee ne portent plus `startDirection`,
  - le moteur interne historique reste disponible pour l'explorateur/recommandations quand il est encore utile.

- [x] `FUNC-P0-03B` (`P0`, `S`) - Smoke tests generation + export GPX.
  Owners: `Routes`, `Back-Go`, `Back-Kotlin`, `QA`.
  Scope:
  - ajouter une fixture partagee `test-fixtures/routes/strava-art-smoke.json`,
  - couvrir generation shape + cache route + export GPX en Go,
  - couvrir generation shape + export GPX en Kotlin,
  - ajouter un smoke manuel `./scripts/manual-strava-art-smoke-check.sh` contre un backend reel avec OSRM,
  - documenter le check dans `docs/routing/checks/strava-art-smoke.md`.
  Acceptance:
  - les tests Go/Kotlin lisent la meme fixture,
  - chaque backend valide au moins un retour de route et un GPX exportable,
  - le smoke manuel echoue clairement si generation ou export GPX casse.

- [ ] `FUNC-P0-03C` (`P0`, `M`) - UX MVP dessin/import/generation.
  Owners: `Product`, `Front`, `Routes`.
  Parcours:
  - choisir un point de depart ou utiliser la position courante,
  - dessiner une forme sur la carte ou importer un GPX,
  - ajuster le dessin avant generation: annuler, effacer, repositionner, simplifier au besoin,
  - choisir le style d'activite: ride, gravel, MTB, run, trail ou hike,
  - snapper la forme au reseau routier OSRM,
  - lancer la generation sans distance cible, denivele cible, direction ou mode de boucle sportive.
  Entrees utilisateur:
  - `shapeInputType`: `draw`, `polyline`, `gpx`, `svg`,
  - `shapeData`: dessin, polyline encodee, GPX importe ou forme,
  - `startPoint`: optionnel mais recommande pour ancrer l'art,
  - `routeType`: `RIDE`, `MTB`, `GRAVEL`, `RUN`, `TRAIL`, `HIKE`,
  - `variantCount`: nombre de propositions souhaitees.
  Acceptance:
  - l'utilisateur peut dessiner ou importer un GPX et lancer une generation,
  - les controles affiches correspondent au mode unique `Draw art`,
  - la carte affiche dessin, point de depart et routes generees sans ambiguite,
  - les propositions Strava Art proviennent du shape mode OSRM et ne reutilisent pas d'anciennes sorties comme routes de remplacement,
  - un candidat trop eloigne du dessin reste affichable avec un score `Art fit` faible et une raison explicite, sans score flatteur ni blocage dur.
  Progress:
  - [x] l'algo preserve la position geographique du dessin quand il est deja place autour du depart, au lieu d'ancrer le premier point du croquis sur le depart,
  - [x] le snapping OSRM utilise davantage d'ancres du dessin et le score visuel penalise plus fortement derive du centre, ordre du trace et ecart de distance,
  - [x] les formes fermees (cercle, etoile, coeur) demarrent le routage sur le contour le plus proche de l'ancre au lieu de relier le centre au contour,
  - [x] en mode Strava Art, une faible ressemblance ne bloque plus la generation: elle degrade `Art fit` et ajoute une raison `below ideal`,
  - [x] fallback OSRM best-effort: si les strategies dessin strictes ne produisent aucun candidat, tenter des waypoints simplifies/enveloppe et retourner une route faible confiance au lieu de bloquer,
  - [x] strategie OSRM `simplified sketch anchors`: essayer en generation normale une version reduite du dessin pour mieux router les formes simples (cercle, etoile, carre) avant le fallback,
  - [x] auto-fit avant routage: option activee par defaut pour recentrer et redimensionner le sketch autour du point de depart et de la carte visible avant l'appel OSRM.

- [x] `FUNC-P0-03D` (`P0`, `M`) - Resultats lisibles et diagnostics produit.
  Owners: `Product`, `Front`, `Routes`.
  Resultats UI:
  - afficher un apercu carte par proposition,
  - afficher distance, D+, duree, score `art fit`/ressemblance, score route/praticabilite,
  - afficher les raisons principales: surface, fallback, snapping, type de route,
  - permettre la selection d'une proposition,
  - permettre l'export GPX de la proposition selectionnee.
  Diagnostics UX:
  - aucune route trouvee pour cette forme,
  - forme trop complexe, trop courte ou trop difficile a snapper,
  - point de depart difficile a router ou deplace vers le point routable le plus proche,
  - candidats historiques ou non-shape ignores quand aucun shape mode OSRM ne respecte le dessin,
  - route type fallback,
  - relaxation anti-backtracking eventuelle,
  - messages comprehensibles dans l'UI, diagnostics techniques conserves pour debug.
  Acceptance:
  - les distances, D+ et durees sont presentes comme resultats, pas comme objectifs,
  - chaque carte de proposition explique pourquoi elle est bonne ou degradee,
  - un echec de generation est actionnable sans lire les logs.

- [x] `FUNC-P0-03E` (`P1`, `M`) - Comparaison visuelle dessin original vs route generee.
  Owners: `Product`, `Front`, `Routes`.
  Scope:
  - comparaison visuelle avant/apres entre dessin original et route OSRM,
  - superposer dessin et route sur la carte avec styles differencies,
  - aider a comprendre les ecarts de snapping.
  Acceptance:
  - l'utilisateur voit immediatement si la route conserve la forme de depart,
  - les ecarts majeurs de ressemblance sont visibles sans ouvrir un detail technique.
  Progress:
  - [x] validation visuelle produit `Route follows sketch`: bon/moyen/faible avec explication courte basee sur `Art fit`.

- [x] `FUNC-P0-03F` (`P1`, `M`) - Outils de transformation du dessin.
  Owners: `Product`, `Front`.
  Scope:
  - deplacer, scale, rotation, recentrage autour du depart,
  - lissage et simplification,
  - annuler/refaire les operations de transformation.
  Acceptance:
  - l'utilisateur peut ajuster une forme importee/dessinee sans recommencer de zero.

- [x] `FUNC-P0-03G` (`P1`, `M`) - Bibliotheque de formes et sauvegarde locale.
  Owners: `Product`, `Front`.
  Scope:
  - bibliotheque locale de formes via combo: coeur, etoile, cercle, carre, triangle, losange, rectangle, hexagone.
  - modeles sauvegardes,
  - sauvegarde locale des creations et export PNG de l'apercu,
  - mode freestyle pour traces non strictement snappees en parc ou terrain ouvert.
  Acceptance:
  - l'utilisateur peut repartir d'un modele ou reprendre une creation precedente.

- [ ] `FUNC-P0-03H` (`P2`, `L`) - Exports avances et assistant de correction.
  Owners: `Product`, `Routes`, `Front`, `Back-Go`, `Back-Kotlin`.
  Scope:
  - [x] generation par prompt simple,
  - [x] import image a tracer,
  - [x] galerie personnelle locale de templates,
  - [ ] galerie publique de templates,
  - [x] export TCX en plus du GPX,
  - [ ] export FIT binaire,
  - [x] assistant de correction: ameliorer la ressemblance, reduire distance, recuperer un echec OSRM, avec affichage proche du canvas carte,
  - [ ] comparaison post-activite entre la route prevue et l'activite reellement enregistree.
  Acceptance:
  - l'utilisateur dispose d'options avancees sans alourdir le MVP.

- [ ] `TECH-P1-03` (`P1`, `M`) - Industrialisation technique Strava Art.
  Owners: `Routes`, `Back-Go`, `Back-Kotlin`, `QA`.
  Scope:
  - OSRM reste le moteur principal de snapping,
  - Go et Kotlin restent alignes pour les endpoints routes et diagnostics,
  - les diagnostics techniques restent disponibles, mais l'UI doit les traduire en signaux produit,
  - les contrats API routes doivent converger vers OpenAPI (`TECH-P1-01`),
  - les checks OSRM doivent devenir automatisables (`TECH-P1-02`).
  Acceptance:
  - les contrats Strava Art sont documentes et testables,
  - les checks route peuvent tourner en CI/smoke local sans donnees personnelles,
  - les ecarts Go/Kotlin sont detectes avant regression utilisateur.

### Priorite moyenne

- [ ] `FUNC-P1-04` (`P1`, `M`) - Comparaison d'activite a effort similaire.
  Owners: `Product`, `Stats`, `Front`.
  Proposition:
  - dans le detail activite, comparer avec les sorties proches en distance/D+/sport/saison,
  - afficher ecarts de vitesse, frequence cardiaque, puissance, cadence et segments communs,
  - indiquer si la sortie est atypique.
  Acceptance:
  - une activite donne immediatement du contexte par rapport aux sorties comparables.

- [ ] `FUNC-P1-09` (`P1`, `M`) - Detecteur d'ascensions avec hysteresis.
  Owners: `Product`, `Stats`, `Back-Go`, `Back-Kotlin`, `Front`.
  Constat:
  - le detecteur actuel de `Slope` segmente les montees par changements locaux de pente,
  - les ascensions irregulieres peuvent etre trop fragmentees ou mal bornees dans le detail activite.
  Proposition:
  - basculer vers une detection d'ascensions soutenues avec seuil de demarrage et seuil de sortie distincts,
  - utiliser une pente lissee sur fenetre de distance, idealement `grade_smooth` quand disponible,
  - fusionner les faux-plats courts au milieu d'une meme montee,
  - garder l'affichage principal centre sur les ascensions, avec libelles et tooltips plus explicites que `Slope`.
  Acceptance:
  - les montees affichees dans le detail activite sont moins fragmentees,
  - les bornes debut/fin restent stables sur des profils irreguliers,
  - Go et Kotlin restent alignes via tests ou fixtures partagees.

- [ ] `FUNC-P1-05` (`P1`, `M`) - Enrichir Routes avec difficulte et lisibilite terrain.
  Owners: `Product`, `Routes`, `Front`.
  Proposition:
  - afficher difficulte estimee, surface mix, part inconnue, confiance du profil OSRM et raisons de fallback directement sur la carte,
  - filtrer ou trier par `plus roulant`, `plus chemin`, `moins de demi-tours`, `plus familier`,
  - conserver les diagnostics techniques mais les traduire en signaux produit.
  Acceptance:
  - un utilisateur peut choisir une route sans lire les raisons brutes du moteur.

- [ ] `FUNC-P1-10` (`P1`, `M`) - Previsions de maintenance materiel.
  Owners: `Product`, `Front`, `Stats`.
  Proposition:
  - projeter les prochaines maintenances selon volume recent, distance totale, composant et historique local,
  - afficher une priorisation simple: urgent, bientot, surveiller,
  - lier les alertes aux activites et aux periodes qui consomment le plus le materiel.
  Acceptance:
  - la vue materiel devient proactive et pas seulement descriptive.

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

- [ ] `FUNC-P2-05` (`P2`, `M`) - Sauvegarde et exports portables des donnees locales.
  Owners: `Product`, `Front`, `Back-Go`, `Back-Kotlin`.
  Proposition:
  - exporter en JSON les objectifs annuels, zones cardio, exclusions/corrections data quality, maintenance materiel et preferences locales,
  - documenter les schemas exportes pour pouvoir les reimporter plus tard,
  - garder les exports GPX de routes generes compatibles avec les outils externes.
  Acceptance:
  - les donnees ajoutees par l'application restent portables hors application.

## Dette visible a traiter en premier

- Smoke tests source modes (`TECH-P0-05`).
- Contrat OpenAPI partage (`TECH-P1-01`).
- Fixtures data quality partagees (`TECH-P1-06`).

## Verification conseillee selon le type de changement

- Docs seulement: relecture Markdown.
- Front: `cd front-vue && npm run type-check && npm run test:unit`.
- Back Go: `cd back-go && go test ./...`.
- Back Kotlin: `cd back-kotlin && ./gradlew test`.
- Routes: lancer les tests cibles Go/Kotlin + checks OSRM automatises ou manuels documentes.
