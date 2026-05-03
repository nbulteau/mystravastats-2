# TODO list

## Etat des lieux au 2026-04-28

- Monorepo avec trois surfaces principales: `front-vue`, `back-go`, `back-kotlin`.
- Le frontend Vue 3 couvre dashboard, objectifs annuels, diagnostics, source modes, data quality, charts, heatmap, statistiques, badges, activites, detail activite, segments, carte, materiel et routes.
- Les modes de source `STRAVA`, `FIT` et `GPX` existent dans Go et Kotlin. Leur activation reste principalement une affaire de configuration runtime et de redemarrage backend.
- Le backend Go reste important pour le binaire local; le backend Kotlin reste la reference historique de plusieurs providers et services metier.
- La generation de routes reste la zone la plus sensible: OSRM, anti-retrace, diagnostics, export GPX, parite Go/Kotlin.
- L'onglet routes a ete repositionne en `Strava Art` / GPS drawing studio: dessiner ou importer une forme, la snapper au reseau routier via OSRM, puis exporter un GPX exploitable.
- La qualite des donnees locales FIT/GPX a deja un socle de diagnostics et corrections locales. Le risque suivant est la validation reproductible: fixtures, smoke tests et comparaison avant/apres correction.
- Les modes source `STRAVA` / `FIT` / `GPX` ont maintenant un smoke test reproductible avec fixtures locales anonymes pour Go et Kotlin.
- La couverture frontend, le contrat API partage et la parite Go/Kotlin hors routes restent les meilleurs leviers pour eviter les regressions silencieuses.

## Garde-fous permanents

- Garder Go et Kotlin alignes pour tout changement de generation de routes.
- Ne jamais transformer l'historique en penalite de nouveaute: il doit rester un signal positif de corridors connus.
- Preserver les regles anti-retrace strictes hors zone depart/arrivee pour les routes sportives classiques et l'explorateur interne.
- Garder le comportement de zone depart/arrivee 2 km explicite et teste.
- Preserver `X-Request-Id` et les diagnostics exploitables sur les endpoints de generation.
- Pour `Strava Art`, conserver `/routes` comme URL interne tant qu'aucune migration n'est prevue.
- Pour `Strava Art`, rendre visibles le dessin d'origine, la route OSRM generee, les scores de ressemblance/praticabilite et les raisons de fallback.
- Pour `Strava Art`, le score `Art fit` doit rester centre sur le respect du dessin: proximite ancree, derive du centre, ordre du trace et forme globale.
- Pour `Strava Art`, le point de depart est un indice de placement, pas une contrainte produit forte: pour une forme fermee, generation et scoring doivent rester flexibles sur le point de depart du contour.
- Pour `Strava Art`, les retours sur ses pas sont acceptables quand ils ameliorent nettement la ressemblance au modele utilisateur; l'anti-retrace devient un signal de praticabilite/diagnostic, pas un rejet dur.
- Garder les exports GPX generes compatibles avec Strava, Garmin, Komoot et les outils GPS standards.
- Ne pas changer silencieusement les contrats API: ajouter migration, compatibilite ou tests de contrat.
- Toute reponse JSON issue d'un provider local doit rester serialisable: pas de `NaN`, `Inf`, sentinelle FIT brute ou tableau `null` quand le contrat expose une liste.
- Toute correction locale doit rester reversible et explicite dans les diagnostics.

## Chantiers techniques proposes

### Priorite haute

- [ ] `TECH-P1-09` (`P1`, `M`) - Industrialiser l'enrolement Strava OAuth.
  Owners: `Back-Go`, `Back-Kotlin`, `Front`, `Docs`.
  Constat:
  - la creation de l'application Strava reste manuelle via `https://www.strava.com/settings/api`,
  - l'API Strava expose OAuth et les donnees athletes/activities/routes, mais pas d'endpoint public pour creer ou administrer une application developpeur,
  - MyStravaStats peut automatiser tout l'apres-creation: ecriture `.strava`, ouverture OAuth, callback local, validation des scopes, refresh token et diagnostics.
  Scope:
  - documenter clairement la limite: app Strava manuelle, OAuth local automatisable,
  - fournir un assistant local pour creer `.strava`, lancer OAuth et sauvegarder `.strava-token.json`,
  - durcir OAuth avec `state`, callback local explicite et refresh token persistant,
  - exposer dans les diagnostics/source modes un statut lisible: credentials presents, token present, scopes acceptes, cache utilisable,
  - garder les secrets hors logs et hors git.
  Acceptance:
  - un utilisateur ne saisit `clientId`/`clientSecret` qu'une fois apres creation de l'app Strava,
  - les lancements suivants reutilisent ou refreshent le token sans rouvrir OAuth tant que l'autorisation reste valide,
  - la documentation ne laisse plus croire que la creation de l'app Strava est automatisable par API.
  Avancement 2026-05-03:
  - fait: README racine, docs OAuth et troubleshooting clarifient la limite Strava,
  - fait: `scripts/setup-strava-oauth.mjs` cree `.strava`, lance OAuth, valide `/api/v3/athlete` et sauvegarde `.strava-token.json`,
  - fait: Go et Kotlin reutilisent/refreshent `.strava-token.json` et ajoutent une verification `state`,
  - fait: `/api/source-modes/preview` signale deja l'absence ou la presence du token OAuth dans ses recommandations,
  - reste: statut OAuth structure (scopes, expiration, athlete) et parcours UI `Connecter Strava`.

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
  - transformer les scenarios anti-retrace legacy, retrace permissif Strava Art, direction, surface, fallback et shape tuning en smoke tests automatises,
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

- [ ] `TECH-P1-08` (`P1`, `M`) - Industrialiser Strava Art apres MVP.
  Owners: `Routes`, `Back-Go`, `Back-Kotlin`, `QA`.
  Scope:
  - documenter le contrat routes Strava Art et les diagnostics exposes,
  - formaliser la politique Strava Art `Art fit` d'abord: autoriser les retours sur ses pas quand ils servent le dessin,
  - rattacher les DTO routes au contrat OpenAPI partage (`TECH-P1-01`),
  - brancher les checks OSRM Strava Art sur les smoke tests automatisables (`TECH-P1-02`),
  - garder Go et Kotlin alignes sur generation, propositions, exports et diagnostics.
  Acceptance:
  - les ecarts Go/Kotlin sur Strava Art sont detectes avant regression utilisateur,
  - les checks route peuvent tourner en CI ou en smoke local sans donnees personnelles.

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

- [ ] `FUNC-P0-03` (`P0`, `M`) - Parcours d'enrolement Strava guide.
  Owners: `Product`, `Front`, `Back-Go`, `Back-Kotlin`.
  Proposition:
  - ajouter dans Diagnostics / Source modes un parcours `Connecter Strava` qui guide l'utilisateur jusqu'a `settings/api`, puis reprend la main pour OAuth,
  - afficher les champs attendus de l'app Strava: `Client ID`, `Client Secret`, `Authorization Callback Domain`,
  - verifier `.strava`, le cache, le token, les scopes et le mode actif sans demander a l'utilisateur de lire les logs,
  - proposer une relance OAuth quand le token est absent, expire, revoque ou incomplet.
  Acceptance:
  - un nouvel utilisateur comprend l'etape manuelle Strava et termine l'enrolement depuis l'application,
  - les erreurs courantes (`clientSecret` faux, callback domain, port occupe, scope refuse) deviennent actionnables dans l'UI.
  Avancement 2026-05-03:
  - premiere tranche livree en CLI/docs/backend,
  - prochaine tranche: brancher le parcours dans Diagnostics / Source modes avec un endpoint d'etat OAuth non bloquant.

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

- [ ] `FUNC-P1-11` (`P1`, `S`) - Etudier https://themechanic.bike/fr pour enrichir l'onglet Gear.
  Owners: `Product`, `Front`, `Stats`.
  Proposition:
  - analyser les fonctionnalites et l'UX de themechanic.bike (suivi composants, alertes maintenance, historique de remplacement, kilomegage par piece),
  - identifier les concepts transposables dans l'onglet Gear sans dependance a un service tiers,
  - proposer un backlog de sous-taches issue de cette analyse.
  Acceptance:
  - un compte-rendu d'analyse documente les inspirations retenues et ecartees,
  - les taches retenues sont ajoutees au TODO sous `FUNC-P1-10` ou remplacent sa proposition si chevauchement.

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

- Enrolement Strava OAuth (`TECH-P1-09`, `FUNC-P0-03`).
- Contrat OpenAPI partage (`TECH-P1-01`).
- Fixtures data quality partagees (`TECH-P1-06`).

## Verification conseillee selon le type de changement

- Docs seulement: relecture Markdown.
- Front: `cd front-vue && npm run type-check && npm run test:unit`.
- Back Go: `cd back-go && go test ./...`.
- Back Kotlin: `cd back-kotlin && ./gradlew test`.
- Routes: lancer les tests cibles Go/Kotlin + checks OSRM automatises ou manuels documentes.
