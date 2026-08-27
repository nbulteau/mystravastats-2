# TODO

## Priorité haute

- [x] `ARCH-P1-01` - Stabiliser le refactoring des grands modules avec des tests ciblés et supprimer les wrappers Kotlin temporaires.
- [x] `ARCH-P1-02` - Extraire les présentations Diagnostics/Activité et l'export PNG GPS Art hors des vues Vue.
- [x] `ARCH-P1-03` - Isoler le transport OSRM dans des clients dédiés Go et Kotlin sans modifier les règles de génération.
- [x] `API-P1-01` - Étendre les schémas OpenAPI générés aux routes, réglages de performance et diagnostics de qualité.
- [x] `SEC-P1-01` - Lier les services Docker à la boucle locale et bloquer les mutations web provenant d'origines non autorisées.

## Priorité moyenne

- [ ] `PRODUCT-P2-01` - Livrer un premier parcours vertical d'objectifs sportifs et de suivi de progression.
- [ ] `PRODUCT-P2-02` - Spécifier un indicateur de charge d'entraînement explicable et robuste aux données manquantes.
- [ ] `ARCH-P2-01` - Poursuivre la réduction des modules encore au-dessus du seuil de 1 000 lignes.

- [ ] `DATA-P2-03` - Étendre progressivement le catalogue européen.
  - traiter ensuite l'Autriche, la Slovénie, l'Allemagne, la Belgique, le Royaume-Uni, la Norvège et les Balkans ;
  - privilégier les ascensions cyclistes documentées plutôt qu'une liste exhaustive de cols routiers ;
  - conserver la parité stricte des catalogues Go/Kotlin, les identifiants stables, la source et la date de vérification ;
  - faire passer chaque lot par `python3 scripts/audit-climb-catalog.py --check`.

## Priorité basse

- [ ] `DATA-P3-01` - Réévaluer le versant de la Creueta depuis La Molina.
  - attendre une source cohérente sur la longueur, le dénivelé et les pentes ;
  - ne pas intégrer ce versant tant que les valeurs publiées restent incompatibles.
