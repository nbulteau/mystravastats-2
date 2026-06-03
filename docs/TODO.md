# TODO list

## Etat des lieux au 2026-05-08

- Monorepo avec trois surfaces principales: `front-vue`, `back-go`, `back-kotlin`.
- Le frontend Vue 3 couvre dashboard, objectifs annuels, diagnostics, source modes, data quality, charts, heatmap, statistiques, badges, activites, detail activite, segments, carte, materiel et routes.
- Les modes de source `STRAVA`, `FIT` et `GPX` existent dans Go et Kotlin. Leur activation peut etre sauvegardee depuis Diagnostics dans `.env`, mais reste effective au redemarrage backend.
- Le backend Go reste important pour le binaire local; le backend Kotlin reste la reference historique de plusieurs providers et services metier.
- La generation de routes reste la zone la plus sensible: OSRM, anti-retrace, diagnostics, export GPX, parite Go/Kotlin.
- L'onglet routes a ete repositionne en `GPS Art` / GPS drawing studio: dessiner ou importer une forme, la snapper au reseau routier via OSRM, puis exporter un GPX exploitable.
- La qualite des donnees locales FIT/GPX dispose maintenant d'un corpus partage et de tests miroir Go/Kotlin sur les anomalies principales: valeurs invalides, streams incomplets, GPS aberrant, altitude spike, corrections proposees et impacts avant/apres correction.
- Les modes source `STRAVA` / `FIT` / `GPX` ont un smoke test reproductible avec fixtures locales anonymes pour Go et Kotlin.
- Les risques ouverts les plus visibles sont le contrat API non partage, les parcours frontend peu couverts, la parite Go/Kotlin hors routes/data quality et la fraicheur des indicateurs apres synchronisation.

## Garde-fous permanents

- Garder Go et Kotlin alignes pour tout changement de generation de routes.
- Ne jamais transformer l'historique en penalite de nouveaute: il doit rester un signal positif de corridors connus.
- Preserver les regles anti-retrace strictes hors zone depart/arrivee pour les routes sportives classiques et l'explorateur interne.
- Garder le comportement de zone depart/arrivee 2 km explicite et teste.
- Preserver `X-Request-Id` et les diagnostics exploitables sur les endpoints de generation.
- Pour `GPS Art`, conserver `/routes` comme URL interne tant qu'aucune migration n'est prevue.
- Pour `GPS Art`, rendre visibles le dessin d'origine, la route OSRM generee, les scores de ressemblance/praticabilite et les raisons de fallback.
- Pour `GPS Art`, le score `Art fit` doit rester centre sur le respect du dessin: proximite ancree, derive du centre, ordre du trace et forme globale.
- Pour `GPS Art`, le trace utilisateur est toujours une polyligne point-a-point ordonnee: meme une forme visuellement fermee ne doit pas etre reinterpretee en boucle sportive, retour depart ou contour a point de depart flexible.
- Pour `GPS Art`, le moteur peut tester des poses automatiques du dessin (echelle, rotation, micro-translation) pour trouver une route OSRM plus fidele, mais les diagnostics doivent exposer la transformation retenue.
- Pour `GPS Art`, les retours sur ses pas sont acceptables quand ils ameliorent nettement la ressemblance au modele utilisateur; l'anti-retrace devient un signal de praticabilite/diagnostic, pas un rejet dur.
- Garder les exports GPX generes compatibles avec Strava, Garmin, Komoot et les outils GPS standards.
- Ne pas changer silencieusement les contrats API: ajouter migration, compatibilite ou tests de contrat.
- Toute reponse JSON issue d'un provider local doit rester serialisable: pas de `NaN`, `Inf`, sentinelle FIT brute ou tableau `null` quand le contrat expose une liste.
- Toute correction locale doit rester reversible et explicite dans les diagnostics.
- Toute evolution data quality doit mettre a jour les fixtures partagees et le snapshot attendu si le diagnostic change volontairement.

## Chantiers techniques proposes

### Priorite haute

- [ ] `TECH-P1-09` (`P1`, `L`) - Ajouter un `CompositeActivityProvider` mixte Strava + RideWithGPS + FIT/GPX/TCX.
  Owners: `Back-Go`, `Back-Kotlin`, `Front`, `QA`.
  Constat:
  - les modes source historiques etaient exclusifs via `FIT_FILES_PATH`, `GPX_FILES_PATH` ou `STRAVA_CACHE_PATH`,
  - un fonctionnement mixte permettrait de garder les metadonnees Strava ou RideWithGPS tout en enrichissant ou completant les activites avec des fichiers locaux FIT/GPX/TCX,
  - RideWithGPS expose des `Trips` et `Routes`, mais ne couvre pas exactement les memes concepts que Strava: le provider doit normaliser ces differences sans les masquer,
  - le cache Strava existant doit rester intact: RideWithGPS doit avoir son propre cache local et la fusion doit produire une vue composite ou un cache composite separe.
  Scope:
  - activer automatiquement le mode composite des que plusieurs sources sont configurees,
  - implementer un `CompositeActivityProvider` miroir en Go et Kotlin,
  - ajouter un provider RideWithGPS avec configuration dediee (`RIDEWITHGPS_API_BASE_URL`, credentials OAuth/token, `RIDEWITHGPS_CACHE_PATH`) et cache local separe sur le modele du provider Strava,
  - charger Strava, RideWithGPS, FIT, GPX et TCX sans changer les providers existants,
  - matcher les activites par date, type, distance, duree et trace GPS quand disponible,
  - exposer provenance, confiance de matching, conflits et activites locales non matchees dans les diagnostics,
  - mettre a jour la section `Data source` de la page Status `/diagnostics` pour afficher les sources actives, les caches Strava/RideWithGPS, les imports locaux et les conflits de fusion,
  - ajouter fixtures partagees Strava + RideWithGPS + FIT + GPX + TCX et tests de parite Go/Kotlin.
  Acceptance:
  - une activite Strava peut etre enrichie par un stream FIT/GPX local sans modifier le cache Strava,
  - une activite RideWithGPS peut etre lue depuis son cache local et enrichie par un fichier FIT/GPX/TCX local,
  - une activite locale absente de Strava peut apparaitre en mode union,
  - la section `Data source` de `/diagnostics` ne presente plus le mode mixte comme une source unique et rend visibles provenance, statut des caches et conflits,
  - les divergences de matching sont visibles et testees dans les deux backends.
  Fait V1:
  - `CompositeActivityProvider` ajoute en Go et Kotlin pour Strava + FIT + GPX,
  - selection automatique quand plusieurs sources sont configurees, avec Strava prioritaire quand il est explicitement configure,
  - matching par date, type, distance, duree et point de depart,
  - matching accepte l'offset horaire Europe/Paris attendu a la date de l'activite (`+1h` hiver, `+2h` ete) pour absorber les divergences timezone/DST quand distance/duree/depart concordent,
  - ID Strava conserve quand un match Strava existe; les activites locales non matchees restent en union,
  - stream local FIT/GPX utilise pour enrichir l'activite composite sans modifier les caches sources,
  - diagnostics composite exposes dans `/api/health/details` et section `Data source` de `/diagnostics` mise a jour,
  - import FIT USB Garmin ajoute cote Go: detection `GARMIN_FIT_SOURCE_PATH` ou `/Volumes/.../GARMIN/ACTIVITY`, copie vers `FIT_FILES_PATH/<annee>/`, bouton `Synchronize` dans Status et reload du provider FIT/composite apres import,
  - tests miroir ajoutes sur conservation de l'ID Strava et union locale.
  Reste:
  - provider RideWithGPS et cache local dedie,
  - support TCX,
  - fixtures partagees completes Strava + RideWithGPS + FIT + GPX + TCX.

### Priorite moyenne

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

- [ ] `FUNC-P1-15` (`P1`, `L`) - Edition aimantee des routes generees `GPS Art`.
  Owners: `Product`, `Front`, `Routes`, `Back-Go`, `Back-Kotlin`.
  Statut: MVP implemente; validation produit avec un OSRM local actif a faire.
  Proposition:
  - apres generation d'une proposition, permettre de modifier la route directement sur la carte sans repasser par un dessin libre,
  - afficher des points de controle/de passage de la route generee, deplacables par l'utilisateur,
  - garder chaque modification aimantee au reseau OSRM: un point deplace est d'abord snappe a une route routable, puis les segments voisins sont recalcules via OSRM,
  - ne jamais ecrire de geometrie hors route dans la route finale ou dans le GPX exporte,
  - distinguer visuellement le dessin original, la route generee et la route editee,
  - permettre au minimum: deplacer un point, inserer un point sur un segment, supprimer un point de controle, annuler/refaire, revenir a la proposition initiale,
  - conserver l'ordre point-a-point du trace GPS Art: l'edition ajuste le chemin OSRM entre points ordonnes, elle ne transforme pas la route en boucle sportive,
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
  - diagnostics explicites d'edition ajoutes et presentes dans `GPS Art`,
  - tests Go/Kotlin ajoutes sur succes d'edition et segment OSRM impossible.

- [ ] `FUNC-P1-13` (`P1`, `M`) - Assistant de revue data quality.
  Owners: `Product`, `Front`, `Stats`.
  Proposition:
  - regrouper les anomalies locales par activite, severite, champ et impact statistique,
  - montrer l'effet avant/apres des corrections proposees avant validation,
  - permettre une validation explicite et reversible des corrections sures.
  Acceptance:
  - la data quality devient un workflow de decision, pas seulement un rapport technique.

### Priorite basse



## Verification conseillee selon le type de changement

- Docs seulement: relecture Markdown.
- Front: `cd front-vue && npm run type-check && npm run test:unit`.
- Back Go: `cd back-go && go test ./...`.
- Back Kotlin: `cd back-kotlin && ./gradlew test`.
- Routes: lancer les tests cibles Go/Kotlin + checks OSRM automatises ou manuels documentes.
