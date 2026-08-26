# TODO

## Priorité haute

## Priorité moyenne

- [ ] `DATA-P2-01` - Vérifier les 18 variantes andorranes en attente.
  - compléter profil, coordonnées GPX, source et métriques plausibles ;
  - ajouter un point de passage pour les variantes partageant une grande partie de leur route ;
  - traiter explicitement les identités transfrontalières d'Os de Civís et du Port de Cabús ;
  - retirer chaque variante validée de `manualReviewVariants` dans `docs/data-sources/climb-catalog-sources.json`.

- [ ] `DATA-P2-02` - Résoudre l'audit de classification restant.
  - examiner les 48 rapprochements Climbfinder ambigus sans appliquer de correspondance approximative ;
  - rechercher une source de difficulté fiable pour les 77 versants absents de Climbfinder ;
  - conserver la classification actuelle quand aucune source suffisamment fiable n'existe ;
  - mettre à jour `docs/data-sources/climb-classification-audit.json` après validation.

- [ ] `DATA-P2-03` - Étendre progressivement le catalogue européen.
  - traiter ensuite l'Autriche, la Slovénie, l'Allemagne, la Belgique, le Royaume-Uni, la Norvège et les Balkans ;
  - privilégier les ascensions cyclistes documentées plutôt qu'une liste exhaustive de cols routiers ;
  - conserver la parité stricte des catalogues Go/Kotlin, les identifiants stables, la source et la date de vérification ;
  - faire passer chaque lot par `python3 scripts/audit-climb-catalog.py --check`.

## Priorité basse

- [ ] `DATA-P3-01` - Réévaluer le versant de la Creueta depuis La Molina.
  - attendre une source cohérente sur la longueur, le dénivelé et les pentes ;
  - ne pas intégrer ce versant tant que les valeurs publiées restent incompatibles.
