# TODO

## Priorité haute

- [x] `ARCH-P1-01` - Stabiliser le refactoring des grands modules avec des tests ciblés et supprimer les wrappers Kotlin temporaires.
- [x] `ARCH-P1-02` - Extraire les présentations Diagnostics/Activité et l'export PNG GPS Art hors des vues Vue.
- [x] `ARCH-P1-03` - Isoler le transport OSRM dans des clients dédiés Go et Kotlin sans modifier les règles de génération.
- [x] `API-P1-01` - Étendre les schémas OpenAPI générés aux routes, réglages de performance et diagnostics de qualité.
- [x] `API-P1-02` - Fiabiliser les 52 opérations OpenAPI : paramètres, corps JSON, formats et statuts ; validation automatique du document et suppression des chemins modèles en doublon.
- [x] `API-P1-03` - Ajouter des scénarios HTTP partagés Go/Kotlin et aligner les erreurs 400/404/405/500, `Allow`, les échecs de synchronisation et `X-Request-Id`, sans changement des URL publiques.
- [x] `SEC-P1-01` - Lier les services Docker à la boucle locale et bloquer les mutations web provenant d'origines non autorisées.

- [x] `DEPS-P1-01` - Actualiser les deux backends et le frontend (2026-10-06) : Go 1.27.1, Kotlin 2.4.20, Gradle 9.8.0, Node 26.10.0, Vue 3.5.43, Vite 8.3.3 et Vitest 5.0.3 ; aligner Docker, CI et scripts de build.

## Priorité moyenne

- [ ] `API-P2-01` - Compléter les schémas de réponse métier et les fixtures de succès ; harmoniser la négociation de contenu et les limites de taille des corps entre backends.
  - [x] Décrire les activités détaillées, séries temporelles, efforts/comparaisons, réglages et estimations FTP, statistiques, records, analyses cardiaques et segments ; préciser unités et valeurs indisponibles.
  - [x] Vérifier 18 fixtures de réponses Go/Kotlin et leurs JSON réels contre OpenAPI dans la CI ; conserver les deux graphies des puissances 20/60 minutes et corriger le plantage Go sans altitude.
  - [ ] Étendre les fixtures aux efforts/comparaisons non vides et aux estimations FTP disponibles, puis aux autres familles de réponses et à la négociation de contenu.
- [ ] `API-P2-02` - Préparer une migration explicite vers des collections paginées et des ressources de synchronisation/génération asynchrones, avec adaptation du frontend et compatibilité des anciennes URL.

- [ ] `PRODUCT-P2-02` - Livrer la charge d'entraînement et les tendances de forme.
  - [x] Préserver les échantillons de puissance absents dans les deux backends, les imports FIT, le stockage JSON et les graphiques ; exclure les fenêtres incomplètes, conserver les zéros mesurés et invalider les résultats en cache lorsque les échantillons changent. Fixtures partagées de calcul et de réponses API.
  - [x] Calculer les fenêtres de puissance sur la durée exacte et les moyennes/zones selon les horodatages réels ; intégrer la puissance normalisée et l’énergie sur le temps couvert. Règle commune Go/Kotlin/front : mesure maintenue jusqu’à la suivante, deux bornes valides, interruption au-delà de 10 s, aucune extrapolation finale. Fixtures partagées irrégulières, lacunaires et rééchantillonnées.
  - [x] Afficher la charge par activité, jour et semaine dans « Progress → Training load », avec comparaison aux quatre semaines précédentes et méthode documentée.
  - [ ] Ajouter les tendances de forme et de fatigue sur une série quotidienne suffisamment couverte ;
  - [x] Détailler chaque contribution, la provenance de la puissance et les données manquantes ; séparer les totaux mesurés/estimés, conserver une charge inconnue à `null`, distinguer absence d’activité enregistrée et repos.
  - [x] Utiliser uniquement la dernière FTP manuelle datée applicable à l’activité pour la charge ; aligner la fiche d’activité et le bilan, sans appliquer rétroactivement la FTP actuelle.
  - [x] Vérifier les deux endpoints sur sept fixtures HTTP partagées et OpenAPI : historique FTP, sources, vrais zéros, lacunes, semaines interannuelles et données sans date. Les activités Strava/FIT/GPX restent visibles ; un score exige une série de puissance exploitable.

- [ ] `PRODUCT-P2-03` - Livrer des courbes de puissance comparées.
  - comparer une activité aux six dernières semaines, à une saison et à l'historique complet ;
  - afficher les meilleurs efforts par durée en W et en W/kg lorsque le poids est disponible, en explicitant le poids utilisé ;
  - relier chaque record à son activité et à l'intervalle correspondant ;
  - réutiliser les calculs temporels fiabilisés et signaler les périodes insuffisamment couvertes, avec des tests de parité Go/Kotlin.

- [ ] `PRODUCT-P2-04` - Livrer un bilan hebdomadaire explicable.
  - [x] Première version intégrée à Training load : volume/dénivelé, intensité mesurée, charge, meilleur effort mesuré de 5 min et quatre semaines de référence ; explications déterministes reliées aux activités et aux données manquantes.
  - [ ] Étendre les records de la semaine aux événements de records personnels sur toutes les durées/métriques.
  - synthétiser le volume, le dénivelé, la répartition de l'intensité, la charge et les records, avec comparaison aux semaines précédentes ;
  - produire d'abord des conclusions déterministes, chacune reliée aux chiffres, à la période de référence et aux activités qui la justifient ;
  - expliciter les données manquantes et les estimations, sans présenter les tendances comme un diagnostic médical ;
  - s'appuyer sur les indicateurs de charge et de puissance validés ci-dessus, sans dépendance à des objectifs sportifs ni à une IA générative.

- [ ] `ARCH-P2-01` - Poursuivre la réduction des modules encore au-dessus du seuil de 1 000 lignes.

- [ ] `DATA-P2-03` - Étendre progressivement le catalogue européen.
  - traiter ensuite l'Autriche, la Slovénie, l'Allemagne, la Belgique, le Royaume-Uni, la Norvège et les Balkans ;
  - privilégier les ascensions cyclistes documentées plutôt qu'une liste exhaustive de cols routiers ;
  - conserver la parité stricte des catalogues Go/Kotlin, les identifiants stables, la source et la date de vérification ;
  - faire passer chaque lot par `python3 scripts/audit-climb-catalog.py --check`.

- [ ] `DEPS-P2-01` - Réévaluer TypeScript 7 lorsque la chaîne Vue/`vue-tsc` prend en charge le compilateur natif.
- [ ] `DEPS-P2-02` - Mettre à jour la chaîne ESLint dès publication du correctif `braces` (GHSA-vfj7-8cjw-p6xm) ; quatre alertes npm limitées aux dépendances de développement.

## Priorité basse

- [ ] `DATA-P3-01` - Réévaluer le versant de la Creueta depuis La Molina.
  - attendre une source cohérente sur la longueur, le dénivelé et les pentes ;
  - ne pas intégrer ce versant tant que les valeurs publiées restent incompatibles.
