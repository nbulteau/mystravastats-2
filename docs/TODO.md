## Proposition Claude Sonnet

### Améliorations techniques

#### 1. Fragmentation du store Pinia monolithique

Le fichier `front-vue/src/stores/context.ts` (269 lignes) est un *god store* qui concentre l'intégralité de l'état applicatif : statistiques, activités, graphiques, carte, badges, dashboard, zones de fréquence cardiaque et gestion des toasts.

**Problèmes actuels :**
- Un changement d'année ou de type d'activité déclenche `updateData()` qui recharge toutes les données du domaine concerné en bloc, même celles déjà à jour.
- La cohabitation de domaines fonctionnels très différents dans un seul store rend la lecture, les tests et les évolutions difficiles.
- Les recomputations de getters Pinia sont globales : un changement de `currentView` invalide l'ensemble des dépendances réactives.

**Proposition :**
Découper `context.ts` en stores domaine indépendants : `useAthleteStore`, `useStatisticsStore`, `useActivitiesStore`, `useChartsStore`, `useDashboardStore`, `useBadgesStore`. Chaque store expose son propre état, ses actions de chargement et ses getters. Le store racine ne conserve plus que `currentYear`, `currentActivityType` et `currentView`. Cette refactorisation suit le principe de responsabilité unique, améliore la testabilité et réduit les renders inutiles.

---

#### 2. Persistance du `BestEffortCache` entre les redémarrages

La classe `BestEffortCache` (`domain/services/statistics/BestEffortCache.kt`) est un `ConcurrentHashMap` purement en mémoire, vidé à chaque arrêt de l'application.

**Problèmes actuels :**
- Le calcul des meilleurs efforts (fenêtre glissante sur les streams de toutes les activités) est CPU-intensif et est intégralement rejoué à chaque démarrage.
- Avec un historique dense (plusieurs centaines d'activités, chacune avec un stream de distance/altitude/vitesse), le premier appel à `/api/statistics` peut prendre plusieurs secondes.
- Le cache est invalidé globalement (`clear()`) même lors d'un rafraîchissement partiel (ajout d'une seule année).

**Proposition :**
Sérialiser le cache sur disque dans le répertoire `strava-cache` (format JSON ou binaire via kotlinx.serialization) avec une clé composite `(activityId, metric, target, streamSize)` déjà présente dans `EffortCacheKey`. Ajouter une invalidation ciblée par `activityId` pour éviter de purger l'ensemble lors d'une synchronisation partielle. Le gain de latence au démarrage serait significatif pour les athlètes ayant un long historique Strava.

---

#### 3. Couverture de tests insuffisante sur les algorithmes de calcul

L'arborescence `src/test/` est très peu peuplée au regard de la complexité des algorithmes exposés (nombre d'Eddington, fenêtre glissante pour les best efforts, calcul du gradient optimal, timeline des records personnels).

**Problèmes actuels :**
- Les statistiques de type `BestEffortDistanceStatistic`, `EddingtonStatistic` ou `BestElevationDistanceStatistic` ne sont couvertes par aucun test unitaire visible, alors que ce sont des invariants métier critiques.
- Un bug de régression dans la fenêtre glissante (ex. calcul sur streams incomplets ou activités sans données d'altitude) ne serait détecté qu'en production.
- L'absence de fixtures de streams standardisées complique l'ajout de tests ultérieurs.

**Proposition :**
Créer un module de tests unitaires dédié aux statistiques avec des jeux de données synthétiques (streams courts et contrôlés) pour chaque famille de calcul. Vérifier les cas limites : stream vide, activité sans données GPS, distance cible supérieure à la longueur du stream, égalité parfaite sur le nombre d'Eddington. Ajouter des tests de non-régression sur la sérialisation/désérialisation du cache Strava (lecture de fichiers JSON réels issus du répertoire `strava-cache`).

---

### Améliorations fonctionnelles

#### 1. Analyse de la charge d'entraînement (Training Load)

Actuellement, l'application calcule des métriques d'effort ponctuel (best efforts, records) mais ne propose pas de vision longitudinale de la charge d'entraînement cumulée.

**Proposition :**
Ajouter un indicateur de charge hebdomadaire et mensuelle inspiré du modèle CTL/ATL/TSB (Chronic Training Load / Acute Training Load / Training Stress Balance), calculable à partir des données disponibles : durée en mouvement, dénivelé, fréquence cardiaque (zones déjà calculées) et puissance (si disponible). Visualiser ces courbes dans l'onglet *Charts* pour permettre à l'athlète d'identifier des périodes de surcharge ou de sous-entraînement. Le backend dispose déjà des streams de fréquence cardiaque et des données de puissance, les ingrédients sont en place.

---

#### 2. Objectifs annuels et projections de fin d'année

L'application affiche l'historique des performances mais ne permet pas à l'athlète de se fixer des objectifs et de visualiser sa progression vers ceux-ci.

**Proposition :**
Ajouter dans la vue *Dashboard* un bloc "Objectifs de l'année" où l'athlète définit des cibles (distance totale, dénivelé total, nombre d'Eddington cible, nombre de sorties). Pour chaque objectif, afficher :
- la progression actuelle (barre de progression + pourcentage),
- la date estimée d'atteinte basée sur la tendance des dernières semaines,
- un indicateur visuel (en avance / dans les temps / en retard) par rapport au rythme nécessaire.

Les objectifs seraient persistés dans le répertoire `strava-cache` (fichier JSON local par athlète), sans dépendance à Strava.

---

#### 3. Comparaison de deux périodes ou de deux athlètes

L'application est centrée sur un seul athlète et une seule année à la fois. Il n'est pas possible de comparer directement deux saisons ou deux pratiquants partageant le même serveur.

**Proposition :**
- **Comparaison de périodes :** ajouter un mode "comparer" dans les vues *Statistics* et *Dashboard* permettant de juxtaposer deux années (ex. 2024 vs 2025) sur les mêmes métriques. La vue *Charts* cumulative par année réalise déjà une partie de ce travail visuellement ; il s'agirait d'étendre cette logique à l'ensemble des statistiques et au dashboard.
- **Comparaison multi-athlètes :** le backend Kotlin dispose déjà de providers GPX/FIT et Strava paramétrables. Exposer un endpoint `/api/compare?athletes=A,B&metric=...` permettrait de générer des tableaux comparatifs entre plusieurs caches locaux, utile dans un contexte de groupe (famille, club).


## Proposition GPT 5.3 Codex

### Améliorations techniques

#### 1. Contrat API typé et génération de clients partagés

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

#### 2. Warmup de cache piloté par priorité et résumés pré-calculés

**Contexte / problème :**
La première utilisation après import Strava peut rester lente : certaines vues critiques (statistiques globales, dashboard annuel) attendent des calculs coûteux et des streams encore froids.

**Proposition concrète :**
Introduire un pipeline de warmup asynchrone dans `back-kotlin` :
- priorité 1 : métriques globales et dashboard,
- priorité 2 : best efforts majeurs (1 km, 5 km, 20 min, 1 h),
- priorité 3 : métriques avancées.
Persister des résumés annuels pré-calculés dans `strava-cache` avec version de schéma, et invalidation ciblée par année lors d'un refresh partiel.

**Valeur attendue :**
Temps de réponse perçu plus stable, navigation initiale plus fluide, et baisse de charge CPU sur les redémarrages fréquents.

---

#### 3. Observabilité applicative et diagnostics guidés

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

#### 1. Plan d'entraînement adaptatif basé sur l'historique réel

**Contexte / problème :**
Les statistiques actuelles décrivent bien le passé, mais proposent peu d'aide prescriptive pour la suite (quoi faire cette semaine pour progresser sans surcharger).

**Proposition concrète :**
Créer un module "Plan adaptatif" qui suggère des volumes hebdomadaires par sport selon la tendance récente (charge, récupération, fréquence des sorties) et les objectifs choisis. Le module génère des recommandations simples : semaine allégée, maintien, ou progression.

**Valeur attendue :**
Passage d'une app descriptive à une app d'aide à la décision, avec un usage plus régulier entre deux sorties.

---

#### 2. Détection d'anomalies de performance et d'adhérence

**Contexte / problème :**
Quand la performance baisse ou que la routine change (forte chute de volume, dérive cardiaque inhabituelle), l'utilisateur doit aujourd'hui le détecter lui-même via les graphiques.

**Proposition concrète :**
Ajouter des alertes non bloquantes dans *Dashboard* :
- baisse anormale de distance/temps sur 3 à 6 semaines,
- hausse du ratio "hard" sans récupération,
- rupture de régularité par rapport aux habitudes historiques.
Chaque alerte inclut une explication courte et une vue détaillée liée.

**Valeur attendue :**
Détection précoce des phases de fatigue ou de démotivation, avec des insights actionnables plutôt qu'une simple visualisation passive.

---

#### 3. Explorateur d'itinéraires personnels et recommandations de sorties

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

## Proposition Gemini 2.5 Pro

### Améliorations techniques

#### 1. Migration vers une base de données embarquée pour le cache

**Contexte / problème :**
Le cache actuel est basé sur des fichiers JSON et des fichiers bruts dans le répertoire `strava-cache`. Cette approche, bien que simple, présente des limites en termes de performance sur les requêtes complexes, de concurrence d'accès et de cohérence des données, notamment pour des calculs transversaux comme les "best efforts" ou les statistiques agrégées.

**Proposition concrète :**
Remplacer le cache basé sur des fichiers par une base de données embarquée légère comme H2, SQLite ou DuckDB.
- **Schéma :** Définir un schéma de base de données pour stocker les athlètes, les activités, les streams et les statistiques pré-calculées.
- **Migration :** Créer un mécanisme de migration qui peuple la base de données à partir des fichiers de cache existants ou d'une nouvelle synchronisation Strava.
- **Accès :** Remplacer les `LocalRepository` basés sur des fichiers par des implémentations basées sur JDBC ou un ORM léger (par exemple, jOOQ, Exposed pour Kotlin).

**Valeur attendue :**
- **Performance :** Accélération significative des requêtes qui nécessitent de scanner de nombreuses activités (ex: calcul du nombre d'Eddington, recherche de PRs).
- **Robustesse :** Meilleure gestion de la concurrence et des transactions, réduisant les risques de corruption du cache.
- **Flexibilité :** Facilité d'ajout de nouvelles requêtes et d'index pour supporter de futures fonctionnalités analytiques sans ré-architecturer l'accès aux données.

---

#### 2. Modularisation du backend Kotlin par fonctionnalité

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

#### 3. Mise en place d'un "Design System" partagé pour le frontend

**Contexte / problème :**
Le frontend Vue est composé de nombreuses vues et composants. Sans un système de design formalisé, il y a un risque d'incohérence visuelle (couleurs, espacements, typographie) et de duplication de code pour des éléments d'interface utilisateur similaires.

**Proposition concrète :**
Créer un "Design System" ou une bibliothèque de composants d'interface utilisateur réutilisables.
- **Composants de base :** Développer un ensemble de composants de base agnostiques de la logique métier (Boutons, Cartes, Modales, Icônes, Entrées de formulaire).
- **Tokens de design :** Centraliser les variables de style (couleurs, polices, espacements) dans des fichiers CSS ou des variables de pre-processeur (Sass/Less).
- **Storybook :** Utiliser un outil comme Storybook pour documenter, visualiser et tester les composants en isolation.
- **Intégration :** Remplacer progressivement les composants ad-hoc dans l'application par ceux de la bibliothèque partagée.

**Valeur attendue :**
- **Consistance :** Une interface utilisateur plus cohérente et professionnelle.
- **Efficacité :** Accélération du développement de nouvelles fonctionnalités en réutilisant des composants existants et éprouvés.
- **Maintenance :** Mises à jour de style plus faciles en modifiant les tokens de design à un seul endroit.

---

### Améliorations fonctionnelles

#### 1. Analyse et suivi du matériel (chaussures, vélos)

**Contexte / problème :**
Les athlètes utilisent différents équipements (plusieurs paires de chaussures pour la course, différents vélos pour la route ou le VTT) qui ont une durée de vie limitée. L'application ne permet pas de suivre l'usure de cet équipement.

**Proposition concrète :**
Ajouter une section "Matériel" permettant à l'utilisateur de :
- **Enregistrer son équipement :** Ajouter des vélos, des chaussures, etc., en spécifiant un nom, une marque, un modèle et une distance d'alerte (ex: 800 km pour des chaussures).
- **Associer l'équipement aux activités :** Permettre d'associer une ou plusieurs pièces d'équipement à chaque activité (possiblement en récupérant l'information de Strava si elle existe).
- **Visualiser l'usure :** Afficher un tableau de bord montrant la distance totale parcourue avec chaque équipement, une barre de progression par rapport à la distance d'alerte, et des alertes visuelles lorsque l'équipement approche de sa fin de vie.

**Valeur attendue :**
- **Utilitaire pratique :** Aide l'athlète à gérer la rotation de son matériel et à prévenir les blessures ou les pannes liées à l'usure.
- **Engagement :** Ajoute une raison supplémentaire d'utiliser l'application régulièrement pour maintenir les données à jour.
- **Exploitation des données :** Utilise les données d'activité existantes pour fournir une nouvelle perspective analytique.

---

#### 2. Analyse de la performance en côte (segments)

**Contexte / problème :**
L'application calcule des statistiques de dénivelé et des "famous climbs", mais ne fournit pas une analyse détaillée des performances répétées sur les mêmes côtes, ce qui est un aspect clé de l'entraînement pour de nombreux cyclistes et coureurs.

**Proposition concrète :**
Créer une vue "Analyse de segments" qui :
- **Détecte les efforts répétés :** Identifie automatiquement les efforts réalisés sur les mêmes segments (côtes) à travers l'historique des activités (basé sur la correspondance des coordonnées GPS).
- **Affiche l'historique des performances :** Pour chaque segment identifié, affiche un tableau et un graphique montrant l'évolution des temps, de la puissance moyenne et de la fréquence cardiaque moyenne au fil du temps.
- **Classement personnel :** Affiche le record personnel (PR) et le top 3 des efforts sur chaque segment.
- **Filtres :** Permet de filtrer par nom de segment, par date, ou par sport.

**Valeur attendue :**
- **Analyse de progression ciblée :** Permet à l'athlète de voir concrètement ses progrès sur ses parcours d'entraînement favoris.
- **Motivation :** Met en évidence les records personnels et encourage à battre ses propres temps.
- **Valorisation des données GPS :** Exploite plus en profondeur les données de streams GPS déjà présentes dans le cache.

---

#### 3. Tableau de bord de la santé et de la récupération

**Contexte / problème :**
L'application se concentre sur les métriques de performance (vitesse, distance, puissance), mais offre peu d'indicateurs sur l'état de forme, la fatigue ou la récupération de l'athlète, qui sont cruciaux pour un entraînement durable.

**Proposition concrète :**
Ajouter un "Tableau de bord Santé" qui synthétise des indicateurs de récupération et de charge.
- **Fréquence cardiaque au repos :** Permettre à l'utilisateur de saisir manuellement sa fréquence cardiaque au repos chaque matin. Un graphique montrerait la tendance, une hausse pouvant indiquer une fatigue.
- **Variabilité de la fréquence cardiaque (VFC/HRV) :** Si l'utilisateur enregistre cette donnée avec un autre appareil, permettre son importation ou sa saisie manuelle.
- **Qualité du sommeil :** Permettre la saisie manuelle de la durée et de la qualité perçue du sommeil.
- **Corrélation :** Mettre en perspective ces indicateurs avec la charge d'entraînement (calculée via le CTL/ATL/TSB proposé par ailleurs) pour aider l'athlète à corréler sa récupération avec ses efforts.

**Valeur attendue :**
- **Vision holistique de l'entraînement :** Passe d'une simple analyse de performance à un outil d'aide à la gestion de l'équilibre entre entraînement et récupération.
- **Prévention du surentraînement :** Fournit des signaux d'alerte précoces en cas de fatigue accumulée.
- **Responsabilisation de l'athlète :** Encourage l'utilisateur à être plus à l'écoute de son corps.
