
### Améliorations techniques

#### Migration vers une base de données embarquée pour le cache

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

#### Modularisation du backend Kotlin par fonctionnalité

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

#### Mise en place d'un "Design System" partagé pour le frontend

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

#### Contrat API typé et génération de clients partagés

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
#### Observabilité applicative et diagnostics guidés

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

#### Analyse de la charge d'entraînement (Training Load)

Actuellement, l'application calcule des métriques d'effort ponctuel (best efforts, records) mais ne propose pas de vision longitudinale de la charge d'entraînement cumulée.

**Proposition :**
Ajouter un indicateur de charge hebdomadaire et mensuelle inspiré du modèle CTL/ATL/TSB (Chronic Training Load / Acute Training Load / Training Stress Balance), calculable à partir des données disponibles : durée en mouvement, dénivelé, fréquence cardiaque (zones déjà calculées) et puissance (si disponible). Visualiser ces courbes dans l'onglet *Charts* pour permettre à l'athlète d'identifier des périodes de surcharge ou de sous-entraînement. Le backend dispose déjà des streams de fréquence cardiaque et des données de puissance, les ingrédients sont en place.

---

#### Objectifs annuels et projections de fin d'année

L'application affiche l'historique des performances mais ne permet pas à l'athlète de se fixer des objectifs et de visualiser sa progression vers ceux-ci.

**Proposition :**
Ajouter dans la vue *Dashboard* un bloc "Objectifs de l'année" où l'athlète définit des cibles (distance totale, dénivelé total, nombre d'Eddington cible, nombre de sorties). Pour chaque objectif, afficher :
- la progression actuelle (barre de progression + pourcentage),
- la date estimée d'atteinte basée sur la tendance des dernières semaines,
- un indicateur visuel (en avance / dans les temps / en retard) par rapport au rythme nécessaire.

Les objectifs seraient persistés dans le répertoire `strava-cache` (fichier JSON local par athlète), sans dépendance à Strava.

#### Plan d'entraînement adaptatif basé sur l'historique réel

**Contexte / problème :**
Les statistiques actuelles décrivent bien le passé, mais proposent peu d'aide prescriptive pour la suite (quoi faire cette semaine pour progresser sans surcharger).

**Proposition concrète :**
Créer un module "Plan adaptatif" qui suggère des volumes hebdomadaires par sport selon la tendance récente (charge, récupération, fréquence des sorties) et les objectifs choisis. Le module génère des recommandations simples : semaine allégée, maintien, ou progression.

**Valeur attendue :**
Passage d'une app descriptive à une app d'aide à la décision, avec un usage plus régulier entre deux sorties.

---

#### Explorateur d'itinéraires personnels et recommandations de sorties

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