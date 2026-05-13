# TODO list

## Etat des lieux au 2026-05-08

- Monorepo avec trois surfaces principales: `front-vue`, `back-go`, `back-kotlin`.
- Le frontend Vue 3 couvre dashboard, objectifs annuels, diagnostics, source modes, data quality, charts, heatmap, statistiques, badges, activites, detail activite, segments, carte, materiel et routes.
- Les modes de source `STRAVA`, `FIT` et `GPX` existent dans Go et Kotlin. Leur activation reste principalement une affaire de configuration runtime et de redemarrage backend.
- Le backend Go reste important pour le binaire local; le backend Kotlin reste la reference historique de plusieurs providers et services metier.
- La generation de routes reste la zone la plus sensible: OSRM, anti-retrace, diagnostics, export GPX, parite Go/Kotlin.
- L'onglet routes a ete repositionne en `Strava Art` / GPS drawing studio: dessiner ou importer une forme, la snapper au reseau routier via OSRM, puis exporter un GPX exploitable.
- La qualite des donnees locales FIT/GPX dispose maintenant d'un corpus partage et de tests miroir Go/Kotlin sur les anomalies principales: valeurs invalides, streams incomplets, GPS aberrant, altitude spike, corrections proposees et impacts avant/apres correction.
- Les modes source `STRAVA` / `FIT` / `GPX` ont un smoke test reproductible avec fixtures locales anonymes pour Go et Kotlin.
- Les risques ouverts les plus visibles sont le contrat API non partage, les parcours frontend peu couverts, la parite Go/Kotlin hors routes/data quality et la fraicheur des indicateurs apres synchronisation.

## Garde-fous permanents

- Garder Go et Kotlin alignes pour tout changement de generation de routes.
- Ne jamais transformer l'historique en penalite de nouveaute: il doit rester un signal positif de corridors connus.
- Preserver les regles anti-retrace strictes hors zone depart/arrivee pour les routes sportives classiques et l'explorateur interne.
- Garder le comportement de zone depart/arrivee 2 km explicite et teste.
- Preserver `X-Request-Id` et les diagnostics exploitables sur les endpoints de generation.
- Pour `Strava Art`, conserver `/routes` comme URL interne tant qu'aucune migration n'est prevue.
- Pour `Strava Art`, rendre visibles le dessin d'origine, la route OSRM generee, les scores de ressemblance/praticabilite et les raisons de fallback.
- Pour `Strava Art`, le score `Art fit` doit rester centre sur le respect du dessin: proximite ancree, derive du centre, ordre du trace et forme globale.
- Pour `Strava Art`, le trace utilisateur est toujours une polyligne point-a-point ordonnee: meme une forme visuellement fermee ne doit pas etre reinterpretee en boucle sportive, retour depart ou contour a point de depart flexible.
- Pour `Strava Art`, le moteur peut tester des poses automatiques du dessin (echelle, rotation, micro-translation) pour trouver une route OSRM plus fidele, mais les diagnostics doivent exposer la transformation retenue.
- Pour `Strava Art`, les retours sur ses pas sont acceptables quand ils ameliorent nettement la ressemblance au modele utilisateur; l'anti-retrace devient un signal de praticabilite/diagnostic, pas un rejet dur.
- Garder les exports GPX generes compatibles avec Strava, Garmin, Komoot et les outils GPS standards.
- Ne pas changer silencieusement les contrats API: ajouter migration, compatibilite ou tests de contrat.
- Toute reponse JSON issue d'un provider local doit rester serialisable: pas de `NaN`, `Inf`, sentinelle FIT brute ou tableau `null` quand le contrat expose une liste.
- Toute correction locale doit rester reversible et explicite dans les diagnostics.
- Toute evolution data quality doit mettre a jour les fixtures partagees et le snapshot attendu si le diagnostic change volontairement.

## Chantiers techniques proposes

### Priorite haute

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

- [ ] `TECH-P1-08` (`P1`, `M`) - Rendre observable la fraicheur des donnees apres synchronisation.
  Owners: `Back-Go`, `Back-Kotlin`, `Front`, `QA`.
  Constat:
  - l'application peut importer de nouvelles activites au demarrage,
  - les indicateurs derives comme l'Eddington number, le dashboard ou les statistiques peuvent rester percus comme obsoletes si l'UI ou les caches ne signalent pas clairement leur version de donnees.
  Scope:
  - exposer une version ou generation de donnees dans `/api/health/details` ou un endpoint equivalent,
  - invalider explicitement les stores frontend quand une synchronisation modifie le corpus d'activites,
  - ajouter un smoke test couvrant import activite -> recalcul statistiques -> UI actualisee.
  Acceptance:
  - apres import d'une activite, les indicateurs derives visibles se mettent a jour sans reload manuel ambigu.

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
  - ajouter des fixtures partagees pour statistiques, badges, dashboard, heatmap, objectifs annuels, source modes, gear analysis, segments et activites detaillees,
  - garder `test-fixtures/data-quality` comme reference pour les diagnostics/corrections locales deja stabilises,
  - comparer au minimum les champs agreges et les cas limites de dates/streams manquants,
  - documenter les divergences acceptees quand une fonctionnalite n'existe que dans un backend.
  Acceptance:
  - la parite critique n'est plus limitee au moteur routes et a la data quality.

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

### Priorite moyenne

- [ ] `FUNC-P1-15` (`P1`, `L`) - Edition aimantee des routes generees `Strava Art`.
  Owners: `Product`, `Front`, `Routes`, `Back-Go`, `Back-Kotlin`.
  Statut: MVP implemente; validation produit avec un OSRM local actif a faire.
  Proposition:
  - apres generation d'une proposition, permettre de modifier la route directement sur la carte sans repasser par un dessin libre,
  - afficher des points de controle/de passage de la route generee, deplacables par l'utilisateur,
  - garder chaque modification aimantee au reseau OSRM: un point deplace est d'abord snappe a une route routable, puis les segments voisins sont recalcules via OSRM,
  - ne jamais ecrire de geometrie hors route dans la route finale ou dans le GPX exporte,
  - distinguer visuellement le dessin original, la route generee et la route editee,
  - permettre au minimum: deplacer un point, inserer un point sur un segment, supprimer un point de controle, annuler/refaire, revenir a la proposition initiale,
  - conserver l'ordre point-a-point du trace Strava Art: l'edition ajuste le chemin OSRM entre points ordonnes, elle ne transforme pas la route en boucle sportive,
  - remonter des diagnostics explicites quand un segment edite ne peut pas etre route par OSRM (`EDIT_SEGMENT_NO_ROUTE`, couverture insuffisante, point non routable),
  - garder Go et Kotlin alignes sur les endpoints/DTO d'edition et les regles de snap.
  Acceptance:
  - un utilisateur peut corriger localement une route orange qui s'eloigne du pointille violet sans redessiner toute la forme,
  - chaque segment edite reste issu du reseau OSRM et l'export GPX reprend la route editee,
  - l'UI montre clairement les parties modifiees et conserve une action de reset vers la route generee,
  - les tests Go/Kotlin couvrent snap de point, reroutage de segment, echec OSRM explicite et preservation de l'ordre point-a-point.
  Fait:
  - contrat `POST /api/routes/{routeId}/edit` ajoute en Go et Kotlin,
  - chaque point de controle est snappe via OSRM nearest puis chaque segment adjacent est recalcule via OSRM route,
  - la route editee est retournee comme nouvelle proposition OSRM et mise en cache pour l'export GPX,
  - l'UI expose le mode edit, points de controle, deplacement, insertion, suppression, undo/redo et reset,
  - diagnostics explicites d'edition ajoutes et presentes dans `Strava Art`,
  - tests Go/Kotlin ajoutes sur succes d'edition et segment OSRM impossible.

- [ ] `FUNC-P1-12` (`P1`, `M`) - Centre de fraicheur et synchronisation.
  Owners: `Product`, `Front`, `Back-Go`, `Back-Kotlin`.
  Proposition:
  - afficher la derniere synchronisation, le nombre d'activites importees, les erreurs provider et la generation de donnees courante,
  - ajouter une action de rafraichissement explicite quand le backend le permet,
  - signaler les vues qui affichent encore des donnees calculees avant la derniere synchronisation.
  Acceptance:
  - l'utilisateur sait si les statistiques visibles incluent les activites nouvellement importees.

- [ ] `FUNC-P1-13` (`P1`, `M`) - Assistant de revue data quality.
  Owners: `Product`, `Front`, `Stats`.
  Proposition:
  - regrouper les anomalies locales par activite, severite, champ et impact statistique,
  - montrer l'effet avant/apres des corrections proposees avant validation,
  - permettre une validation explicite et reversible des corrections sures.
  Acceptance:
  - la data quality devient un workflow de decision, pas seulement un rapport technique.

- [ ] `FUNC-P1-10` (`P1`, `M`) - Previsions de maintenance materiel.
  Owners: `Product`, `Front`, `Stats`.
  Inspiration: [analyse The Bike Mechanic](reference/gear-maintenance-inspiration-themechanic.md).
  Proposition:
  - afficher en tete de l'onglet Gear un tableau de priorite des taches `overdue` / `due` / `soon`, trie par severite et distance/temps restant,
  - expliquer chaque alerte par ses preuves: dernier entretien, odometre au service, distance/temps depuis service, prochaine echeance et regle appliquee,
  - projeter les prochaines maintenances selon le volume mensuel recent du materiel, la distance totale, le composant et l'historique local,
  - regrouper les composants par familles lisibles: transmission, freinage, roues/pneus, suspension, roulements,
  - rendre les roues/pneus plus explicites: pneu avant/arriere, preventif tubeless avant/arriere, obus de valve et voile de roue,
  - distinguer `service` et `remplacement`: un remplacement clot l'ancien cycle et demarre un nouveau cycle au kilometrage courant,
  - ajouter ensuite un inventaire local leger de pieces de rechange, consommable lors d'un remplacement,
  - signaler les limites de prediction quand la couverture d'affectation materiel est faible ou quand le filtre d'annee masque le kilometrage total.
  Acceptance:
  - la vue materiel devient proactive et pas seulement descriptive,
  - le prochain geste de maintenance est visible sans ouvrir chaque velo,
  - les predictions restent locales, explicables et sans dependance a un service tiers.

### Priorite basse

- [ ] `FUNC-P2-02` (`P2`, `M`) - Calendrier d'entrainement unifie.
  Owners: `Product`, `Front`, `Stats`.
  Proposition:
  - vue calendrier combinant heatmap, charge hebdo, jours de repos, sorties longues et intensites,
  - navigation semaine/mois/annee,
  - annotations manuelles locales.
  Acceptance:
  - lecture rapide de la regularite et des trous d'entrainement.

- [ ] `FUNC-P2-05` (`P2`, `M`) - Sauvegarde et exports portables des donnees locales.
  Owners: `Product`, `Front`, `Back-Go`, `Back-Kotlin`.
  Proposition:
  - exporter en JSON les objectifs annuels, zones cardio, exclusions/corrections data quality, maintenance materiel et preferences locales,
  - documenter les schemas exportes pour pouvoir les reimporter plus tard,
  - garder les exports GPX de routes generes compatibles avec les outils externes.
  Acceptance:
  - les donnees ajoutees par l'application restent portables hors application.

- [ ] `FUNC-P2-06` (`P2`, `M`) - Bibliotheque de projets `Strava Art`.
  Owners: `Product`, `Front`, `Routes`.
  Proposition:
  - sauvegarder dessins, imports, routes OSRM generees, exports GPX et scores associes,
  - comparer plusieurs variantes d'un meme dessin,
  - permettre de reprendre un projet sans redessiner depuis zero.
  Acceptance:
  - `Strava Art` devient un atelier reutilisable plutot qu'un outil one-shot.

## Dette visible a traiter en premier

- Contrat OpenAPI partage (`TECH-P1-01`).
- Fraicheur des donnees apres synchronisation (`TECH-P1-08`).
- Couverture frontend des parcours critiques (`TECH-P1-03`).

## Verification conseillee selon le type de changement

- Docs seulement: relecture Markdown.
- Front: `cd front-vue && npm run type-check && npm run test:unit`.
- Back Go: `cd back-go && go test ./...`.
- Back Kotlin: `cd back-kotlin && ./gradlew test`.
- Routes: lancer les tests cibles Go/Kotlin + checks OSRM automatises ou manuels documentes.
