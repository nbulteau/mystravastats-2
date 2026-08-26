# TODO list

## Etat des lieux au 2026-07-23

- Monorepo avec trois surfaces principales: `front-vue`, `back-go`, `back-kotlin`.
- Le frontend Vue 3 couvre dashboard, objectifs annuels, diagnostics, source modes, data quality, charts, heatmap, statistiques, badges, activites, detail activite, segments, carte, materiel et routes.
- Les modes de source `STRAVA`, `FIT` et `GPX` existent dans Go et Kotlin. Leur activation peut etre sauvegardee depuis Diagnostics dans `.env`, mais reste effective au redemarrage backend.
- Le backend Go reste important pour le binaire local; le backend Kotlin reste la reference historique de plusieurs providers et services metier.
- La generation de routes reste la zone la plus sensible: OSRM, anti-retrace, diagnostics, export GPX, parite Go/Kotlin.
- L'onglet routes a ete repositionne en `GPS Art` / GPS drawing studio: dessiner ou importer une forme, la snapper au reseau routier via OSRM, puis exporter un GPX exploitable.
- La qualite des donnees locales FIT/GPX dispose maintenant d'un corpus partage et de tests miroir Go/Kotlin sur les anomalies principales: valeurs invalides, streams incomplets, GPS aberrant, altitude spike, horodatages incoherents, coordonnees invalides et reprise d'enregistrement apres une longue interruption. Les glitches GPS ponctuels sont corriges par interpolation sans supprimer les echantillons capteur alignes.
- Les modes source `STRAVA` / `FIT` / `GPX` ont un smoke test reproductible avec fixtures locales anonymes pour Go et Kotlin.
- En mode source locale ou combinee, les IDs locaux restent compatibles avec les clients JavaScript, les liens Strava ne sont exposes que pour les activites réellement issues de Strava, et la provenance composite detaillee est alignee entre Go et Kotlin.
- Les streams FIT conservent maintenant des series temps/distance/capteurs alignees, y compris sans GPS; les capteurs absents ne sont plus simules par des series de zeros. La fusion combinee reste un appariement un-a-un par fournisseur, trace la provenance des metriques enrichies et remonte les limites de quota Strava dans ses diagnostics.
- Le frontend surveille maintenant en continu l'empreinte du jeu d'activites backend et invalide les caches derives meme si le nombre total d'activites ne change pas ou si le backend redemarre.
- L'onglet Badges peut generer un poster SVG imprimable des cols franchis, avec choix entre trois compositions clairement differenciees: Altitude met les profils en avant dans une mise en page chaleureuse et minimale, Carnet topo utilise une grille technique et une typographie monospaces, et Collection organise des cartes editoriales par pays et massif. Les trois designs acceptent jusqu'a 50 cols, sur une grille de cinq cols par ligne; l'utilisateur peut preselectionner les 50 plus difficiles ou les 50 plus longs. Au-dela de 25 cols, le document passe automatiquement au format 2:3 de 2000 x 3000, adapte a une impression 60 x 90 cm. Le pied de poster resume desormais sans ambiguite le nombre de cols selectionnes et le nombre reel d'ascensions (`… COLS · … ASCENSIONS`); le denivele reste une caracteristique propre a chaque versant et n'est plus additionne comme s'il avait ete parcouru une seule fois. Les DTO Go/Kotlin exposent profils, pays, massif, source, indice de difficulte, caracteristiques, nombre total d'ascensions et uniquement le meilleur temps avec sa date. Les profils ont encore ete agrandis et sont decoupes en surfaces colorees par pente avec kilometres, altitudes intermediaires sans chevauchement et legende; les designs denses Altitude et Carnet topo exploitent maintenant toute la hauteur libre de chaque vignette, et l'altitude de depart est relevee pour ne plus masquer la pente du premier troncon. Dans le design Collection, le profil occupe toute la hauteur disponible et une ligne compacte regroupe distance, denivele, altitude maximale et difficulte sans repeter le nombre d'ascensions. Chaque vignette affiche explicitement `ALT MAX … M · DIFFICULTÉ … PTS`, et les titres utilisent une graisse uniforme sans compression des glyphes. Leurs bornes GPS sont choisies selon la longueur cataloguee pour eviter d'inclure un detour de sortie complet. Une activite ne peut valider qu'un seul versant d'un meme col; des points de passage propres aux variantes proches permettent notamment de distinguer la Madeleine par la D213 de celle via Montgellafrey. Les statistiques cataloguees priment sur un calcul GPS de secours lisse sur 500 m. Les catalogues nationaux synchronises `france.json`, `suisse.json`, `italie.json` et `espagne.json` remplacent les anciens fichiers Alpes/Pyrenees et contiennent 295 sommets / 567 versants: France 154/317, Suisse 23/48, Italie 31/78 et Espagne 87/124. Le catalogue France couvre desormais 15 cols et 29 versants corses, ainsi que les Vosges et davantage d'ascensions du Massif central, du Jura, des Alpes et des Pyrenees. Les catalogues sont controles automatiquement (unicite locale et entre pays, mesures positives, categorie, coordonnees, sources et plausibilite geographique). Les valeurs nouvelles sont recoupees avec cols-cyclisme.com et leurs coordonnees proviennent des GPX de reference; les valeurs de pente cataloguees jusqu'a 30 % sont acceptees, tandis que le calcul GPS de secours reste plafonne a 20 %.
- L'ecran Badges est organise en quatre espaces: recompenses generales, carnet des cols, atelier de posters et carte interactive des cols. Cette carte regroupe les 567 versants en 295 marqueurs sommet, distingue les cols gravis, a decouvrir et favoris, et partage ses filtres et sa navigation avec le carnet. Un resume commun affiche les badges gagnes, les cols reconnus, le nombre d'ascensions et les massifs parcourus.
- En mode de sources combinees, Go et Kotlin utilisent une seule identite de stockage valide pour les corrections de qualite, exclusions, objectifs et autres donnees persistantes: la source Strava est prioritaire, sinon la premiere source locale. Les descriptions composites restent reservees aux diagnostics et ne sont plus interpretees comme des chemins de fichiers Windows.
- Le regroupement journalier du Dashboard Go accepte les timestamps RFC3339 avec offset (`+02:00`) utilises par les FIT; les courbes cumulatives et la heatmap restent ainsi coherentes avec les totaux annuels.
- Les records de vitesse Go/Kotlin ignorent les fenetres distance/temps contenant un segment physiquement impossible et les `maxSpeed` aberrantes; les caches d'efforts ont ete versionnes pour recalculer les activites deja importees.
- Les risques ouverts les plus visibles sont le contrat API non partage, les parcours frontend peu couverts et la parite Go/Kotlin hors routes/data quality.

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
- Une anomalie GPS ponctuelle ne doit pas exclure silencieusement l'activite complete des totaux; seuls les segments ou records contamines doivent etre neutralises.

## Chantiers techniques proposes

### Priorite haute

### Priorite moyenne

### Priorite basse

## Chantiers fonctionnels proposes

### Priorite haute

- [ ] `FUNC-P1-19` (`P1`, `M`) - Stabiliser l'identite sommet/versant et la qualite du catalogue.
  Owners: `Data`, `Back-Go`, `Back-Kotlin`, `QA`.
  Proposition:
  - attribuer un identifiant stable au sommet et un autre a chaque versant, independamment du libelle affiche,
  - documenter les conventions de nommage, les points de depart, les points de passage discriminants et les sources,
  - ajouter des controles sur les doublons semantiques, les profils plats suspects, les deniveles incompatibles avec les altitudes et les pentes maximales aberrantes,
  - produire un rapport de couverture par pays, massif, sommet et nombre de versants attendus/trouves,
  - conserver la possibilite de corriger une variante sans casser l'historique des badges deja attribues.
  Avancement classification:
  - un collecteur reproductible rapproche les 567 versants avec l'API publique Climbfinder sans recalculer l'indice depuis la seule pente moyenne,
  - 490 versants disposent d'une fiche candidate, dont 442 correspondances exactes appliquees avec leur indice Cotacol et leur categorie publies; `SHC` est normalise en `HC`,
  - 48 rapprochements proches restent volontairement a verifier et 77 versants absents de la source conservent leur classification precedente au lieu de recevoir celle d'un col voisin,
  - le rapport `docs/data-sources/climb-classification-audit.json` conserve l'URL, l'identifiant source et les ecarts de depart, sommet, distance et denivele; les trois copies Go/Kotlin/cache sont synchronisees,
  - les regressions Alpe d'Huez (979 points, HC) et Croix-de-Fer depuis Allemond (1 092 points, HC) sont couvertes dans les tests catalogue Go et Kotlin.
  - la tolerance de sommet peut etre resserree par versant lorsque deux routes se separent a proximite immediate: le Glandon depuis Allemond exige un passage a moins de 100 m, ce qui conserve la visite reelle du 12 aout apres la Croix-de-Fer sans compter le simple passage du 19 aout sur la route commune.
  Acceptance:
  - un meme effort ne cree jamais deux badges pour un seul versant,
  - deux vrais versants d'un meme sommet restent selectionnables et comparables separement,
  - les catalogues Go/Kotlin restent identiques et leurs validations sont automatisees,
  - toute correction de profil ou de metrique conserve une source traçable.

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
  Statut: workflow de revue et corrections reversibles implemente; detection des interruptions/reprises ajoutee, segmentation multi-polyligne et map-matching des trous a faire.
  Proposition:
  - regrouper les anomalies locales par activite, severite, champ et impact statistique,
  - montrer l'effet avant/apres des corrections proposees avant validation,
  - permettre une validation explicite et reversible des corrections sures.
  - pour une reprise apres interruption, ne jamais compter ni dessiner comme mesure fiable la ligne droite entre les deux sessions; proposer une coupure de trace ou un map-matching explicitement estime.
  Acceptance:
  - la data quality devient un workflow de decision, pas seulement un rapport technique.

- [ ] `FUNC-P2-23` (`P2`, `M`) - Comparer plusieurs ascensions d'un meme versant.
  Owners: `Product`, `Front`, `Stats`.
  Proposition:
  - superposer progression, temps intermediaires, vitesse, VAM, puissance et frequence cardiaque par distance,
  - comparer par defaut le meilleur temps, la derniere ascension et une ascension choisie,
  - signaler les traces GPS incompletes ou les differences de point de depart qui rendent une comparaison fragile,
  - afficher les gains/pertes par secteur sans transformer une estimation en chronometrage exact.
  Acceptance:
  - seules les ascensions rattachees au meme identifiant de versant sont comparees automatiquement,
  - les donnees absentes n'empechent pas la comparaison des metriques restantes,
  - la methode d'alignement et la precision estimee sont visibles.

- [ ] `FUNC-P2-24` (`P2`, `M`) - Ajouter un tableau de bord du grimpeur.
  Owners: `Product`, `Front`, `Stats`.
  Proposition:
  - calculer altitude cumulee, denivele sur cols, VAM record, plus long versant, col le plus difficile et col le plus gravi,
  - suivre la progression par annee, pays, massif, categorie et tranche d'altitude,
  - separer les statistiques sur tous les cols de celles limitees a la periode selectionnee,
  - permettre d'ouvrir le carnet filtre depuis chaque indicateur.
  Acceptance:
  - chaque statistique renvoie aux ascensions qui la composent,
  - les records ignores par les garde-fous de qualite sont explicites,
  - les totaux restent coherents entre Dashboard, carnet et posters.

- [ ] `FUNC-P2-26` (`P2`, `L`) - Etendre progressivement le catalogue europeen.
  Owners: `Product`, `Data`, `Back-Go`, `Back-Kotlin`, `QA`.
  Statut:
  - France, Suisse, Italie et Espagne sont integrees et synchronisees dans les deux backends,
  - le catalogue Espagne contient 87 sommets / 124 versants sources; quatre variantes frontalieres deja rattachees a `france.json` ne sont pas dupliquees,
  - le versant de la Creueta depuis La Molina reste volontairement exclu tant que ses longueur et pentes publiees sont incoherentes.
  Proposition:
  - consolider France, Suisse, Italie et Espagne avant d'ajouter Andorre, Autriche, Slovenie, Allemagne, Belgique, Royaume-Uni, Norvege et Balkans,
  - prioriser les massifs et ascensions cyclistes documentes plutot qu'une liste exhaustive de cols routiers,
  - exiger pour chaque variante un profil, des metriques plausibles, des coordonnees, une source et si necessaire un point de passage discriminant,
  - publier un rapport de couverture et une liste des variantes a verifier manuellement.
  Acceptance:
  - chaque nouveau catalogue passe les memes validations que `france.json`,
  - les sommets transfrontaliers utilisent des identifiants communs et ne sont pas dupliques,
  - l'origine et la date de verification des donnees sont conservees.

### Priorite basse

## Verification conseillee selon le type de changement

- Docs seulement: relecture Markdown.
- Front: `cd front-vue && npm run type-check && npm run test:unit`.
- Back Go: `cd back-go && go test ./...`.
- Back Kotlin: `cd back-kotlin && ./gradlew test`.
- Routes: lancer les tests cibles Go/Kotlin + checks OSRM automatises ou manuels documentes.
