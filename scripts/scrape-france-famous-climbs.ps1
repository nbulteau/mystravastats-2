param(
    [Parameter(Mandatory = $true)]
    [string] $OutputPath
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"
$invariantCulture = [System.Globalization.CultureInfo]::InvariantCulture
$headers = @{
    "User-Agent" = "Mozilla/5.0 (compatible; MyStravaStats famous-climb catalog refresh)"
}
$cacheDirectory = Join-Path ([System.IO.Path]::GetTempPath()) "mystravastats-cols-cyclisme-cache"
[System.IO.Directory]::CreateDirectory($cacheDirectory) | Out-Null

function Invoke-CatalogRequest {
    param([Parameter(Mandatory = $true)][string] $Url)

    $urlBytes = [System.Text.Encoding]::UTF8.GetBytes($Url)
    $cacheKey = [Convert]::ToHexString([System.Security.Cryptography.SHA256]::HashData($urlBytes))
    $cachePath = Join-Path $cacheDirectory "$cacheKey.txt"
    if ([System.IO.File]::Exists($cachePath)) {
        return [System.IO.File]::ReadAllText($cachePath)
    }

    Start-Sleep -Milliseconds 900
    for ($attempt = 1; $attempt -le 8; $attempt++) {
        $content = & curl.exe `
            --http1.1 `
            --silent `
            --show-error `
            --location `
            --connect-timeout 20 `
            --max-time 60 `
            --user-agent $headers["User-Agent"] `
            $Url
        if ($LASTEXITCODE -eq 0 -and $content) {
            $joinedContent = $content -join "`n"
            [System.IO.File]::WriteAllText(
                $cachePath,
                $joinedContent,
                [System.Text.UTF8Encoding]::new($false)
            )
            return $joinedContent
        }
        if ($attempt -eq 8) {
            throw "Unable to download $Url after $attempt attempts"
        }
        Start-Sleep -Seconds ([math]::Min(5 * $attempt, 30))
    }
}

function ConvertFrom-HtmlCell {
    param([string] $Value)

    return [System.Net.WebUtility]::HtmlDecode(($Value -replace "<[^>]+>", "")).Trim()
}

function Format-StartName {
    param([Parameter(Mandatory = $true)][string] $Name)

    $overrides = @{
        "Saint Maurice sur Moselle" = "Saint-Maurice-sur-Moselle"
        "Saint Amarin" = "Saint-Amarin"
        "Willer sur Thur" = "Willer-sur-Thur"
        "Wihr au Val" = "Wihr-au-Val"
        "Plancher les mines" = "Plancher-les-Mines"
        "Saint Pierre" = "Saint-Pierre"
        "Ville" = "Villé"
        "Luz Saint Sauveur" = "Luz-Saint-Sauveur"
        "Bagneres de Luchon" = "Bagnères-de-Luchon"
        "Ax les Thermes" = "Ax-les-Thermes"
        "Saint Jean le Vieux" = "Saint-Jean-le-Vieux"
        "Saint Lary Soulan" = "Saint-Lary-Soulan"
        "Pierrefitte Nestalas" = "Pierrefitte-Nestalas"
        "Aulus les Bains" = "Aulus-les-Bains"
        "Le Petit Bornand" = "Le Petit-Bornand"
        "Le Bourget du Lac" = "Le Bourget-du-Lac"
        "Chambery" = "Chambéry"
        "Saint Baldoph" = "Saint-Baldoph"
        "Saint Pierre d'Entremont" = "Saint-Pierre-d'Entremont"
        "Saint Jean de Sixt" = "Saint-Jean-de-Sixt"
        "Thones" = "Thônes"
        "Menthon Saint Bernard" = "Menthon-Saint-Bernard"
        "Saint Jean en Royans" = "Saint-Jean-en-Royans"
        "Saint Laurent en Royans" = "Saint-Laurent-en-Royans"
        "Saint Hugues" = "Saint-Hugues"
        "Saint Nazaire les Eymes" = "Saint-Nazaire-les-Eymes"
        "L'Escarene" = "L'Escarène"
        "Saint Etienne les Orgues" = "Saint-Étienne-les-Orgues"
        "Uriage les Bains" = "Uriage-les-Bains"
        "Vaulnaveys le Haut" = "Vaulnaveys-le-Haut"
        "Saint André de Valborgne" = "Saint-André-de-Valborgne"
        "Bourg Argental" = "Bourg-Argental"
        "Saint Chamond" = "Saint-Chamond"
        "Saint Etienne" = "Saint-Étienne"
        "Sainte Eulalie" = "Sainte-Eulalie"
        "Nurieux Volognat" = "Nurieux-Volognat"
        "Serrières sur ain" = "Serrières-sur-Ain"
        "Longevilles Mt d'Or" = "Les Longevilles-Mont-d'Or"
        "Saint Jean de Maurienne" = "Saint-Jean-de-Maurienne"
        "Saint Jean de Maurienne, via Saint Pancrace" = "Saint-Jean-de-Maurienne via Saint-Pancrace"
        "Sainte Foy Tarentaise" = "Sainte-Foy-Tarentaise"
        "L'Argentiere la Bessee" = "L'Argentière-la-Bessée"
        "D 2565" = "D2565"
    }
    if ($overrides.ContainsKey($Name)) {
        return $overrides[$Name]
    }
    return $Name
}

function Get-TableMetric {
    param(
        [Parameter(Mandatory = $true)][string] $Html,
        [Parameter(Mandatory = $true)][string] $Label,
        [switch] $Optional
    )

    $escapedLabel = [regex]::Escape($Label)
    $match = [regex]::Match(
        $Html,
        "<td>\s*$escapedLabel\s*</td>\s*<td[^>]*>(?<value>.*?)</td>",
        [System.Text.RegularExpressions.RegexOptions]::IgnoreCase -bor
            [System.Text.RegularExpressions.RegexOptions]::Singleline
    )
    if (-not $match.Success) {
        if ($Optional) {
            return 0.0
        }
        throw "Metric '$Label' not found"
    }

    $text = ConvertFrom-HtmlCell $match.Groups["value"].Value
    $number = [regex]::Match($text, "-?\d+(?:[\.,]\d+)?")
    if (-not $number.Success) {
        if ($Optional) {
            return 0.0
        }
        throw "Metric '$Label' has no numeric value: $text"
    }

    return [double]::Parse($number.Value.Replace(",", "."), $invariantCulture)
}

function Get-ClimbCategory {
    param([int] $Difficulty)

    if ($Difficulty -ge 900) { return "HC" }
    if ($Difficulty -ge 600) { return "1" }
    if ($Difficulty -ge 300) { return "2" }
    if ($Difficulty -ge 150) { return "3" }
    return "4"
}

function Get-ProfileData {
    param(
        [Parameter(Mandatory = $true)][string] $Url,
        [string] $AlternativeName = ""
    )

    $html = Invoke-CatalogRequest $Url
    $profileIdMatch = [regex]::Match($Url, "-c(?<id>\d+)\.htm$")
    if (-not $profileIdMatch.Success) {
        throw "Unable to determine profile id from $Url"
    }
    $profileId = $profileIdMatch.Groups["id"].Value

    $startMatch = [regex]::Match(
        $html,
        "<td>\s*D(?:é|&eacute;)part\s*:\s*</td>\s*<td[^>]*>(?<value>.*?)</td>",
        [System.Text.RegularExpressions.RegexOptions]::IgnoreCase -bor
            [System.Text.RegularExpressions.RegexOptions]::Singleline
    )
    if (-not $startMatch.Success) {
        throw "Start name not found in $Url"
    }
    $startName = if ($AlternativeName) {
        $AlternativeName
    }
    else {
        Format-StartName (ConvertFrom-HtmlCell $startMatch.Groups["value"].Value)
    }

    $length = Get-TableMetric $html "Longueur :"
    $totalAscent = [int][math]::Round((Get-TableMetric $html "Dénivellation :"))
    $averageGradient = Get-TableMetric $html "% Moyen :"
    $maximumGradient = Get-TableMetric $html "% Maximal :" -Optional
    $topAltitude = [int][math]::Round((Get-TableMetric $html "Altitude :"))

    [xml] $gpx = Invoke-CatalogRequest "https://www.cols-cyclisme.com/gpx/$profileId.gpx"
    $trackPoints = @($gpx.SelectNodes("//*[local-name()='trkpt']"))
    if ($trackPoints.Count -lt 2) {
        throw "GPX $profileId does not contain enough track points"
    }

    $first = $trackPoints[0]
    $last = $trackPoints[-1]
    $firstElevation = [double]::Parse($first.ele, $invariantCulture)
    $lastElevation = [double]::Parse($last.ele, $invariantCulture)
    if ($firstElevation -lt $lastElevation) {
        $summit = $last
        $start = $first
    }
    else {
        $summit = $first
        $start = $last
    }
    $minimumAltitude = [int][math]::Round((
        $trackPoints |
            ForEach-Object { [double]::Parse($_.ele, $invariantCulture) } |
            Measure-Object -Minimum
    ).Minimum)

    $difficulty = [int][math]::Round($length * $averageGradient * $averageGradient)
    return [pscustomobject][ordered]@{
        topAltitude = $topAltitude
        summitCoordinate = [pscustomobject][ordered]@{
            latitude = [double]::Parse($summit.lat, $invariantCulture)
            longitude = [double]::Parse($summit.lon, $invariantCulture)
        }
        midpointCoordinate = [pscustomobject][ordered]@{
            latitude = [double]::Parse($trackPoints[[math]::Floor($trackPoints.Count / 2)].lat, $invariantCulture)
            longitude = [double]::Parse($trackPoints[[math]::Floor($trackPoints.Count / 2)].lon, $invariantCulture)
        }
        alternative = [pscustomobject][ordered]@{
            name = $startName
            geoCoordinate = [pscustomobject][ordered]@{
                latitude = [double]::Parse($start.lat, $invariantCulture)
                longitude = [double]::Parse($start.lon, $invariantCulture)
            }
            length = [math]::Round($length, 2)
            totalAscent = $totalAscent
            minimumAltitude = $minimumAltitude
            maximumGradient = [math]::Round($maximumGradient, 1)
            difficulty = $difficulty
            averageGradient = [math]::Round($averageGradient, 2)
            category = Get-ClimbCategory $difficulty
            sourceUrl = $Url
        }
    }
}

$definitions = @(
    @{ source = "Ballon d'Alsace"; name = "Ballon d'Alsace"; massif = "Vosges" },
    @{ source = "Grand Ballon"; name = "Grand Ballon"; massif = "Vosges" },
    @{ source = "Petit Ballon"; name = "Petit Ballon"; massif = "Vosges" },
    @{ source = "Col de Platzerwasel"; name = "Col du Platzerwasel"; massif = "Vosges" },
    @{ source = "Station de la Planche des Belles Filles"; name = "Planche des Belles Filles"; massif = "Vosges" },
    @{ source = "Col de la Schlucht"; name = "Col de la Schlucht"; massif = "Vosges" },
    @{ source = "Station de Champ du feu"; name = "Champ du Feu"; massif = "Vosges" },

    @{ source = "Col du Vergio"; name = "Col du Vergio"; massif = "Corse" },
    @{ source = "Col de Bavella"; name = "Col de Bavella"; massif = "Corse" },
    @{ source = "Col de Vizzavona"; name = "Col de Vizzavona"; massif = "Corse" },
    @{ source = "Col de l'Ospedale"; name = "Col de l'Ospedale"; massif = "Corse" },
    @{ source = "Col de Scalella"; name = "Col de Scalella"; massif = "Corse" },
    @{ source = "Gorges de la Restonica"; name = "Gorges de la Restonica"; massif = "Corse" },
    @{ source = "Col de Verde"; name = "Col de Verde"; massif = "Corse" },
    @{ source = "Col de Sorba"; name = "Col de Sorba"; massif = "Corse" },
    @{ source = "Col de Prato"; name = "Col de Prato"; massif = "Corse" },
    @{ source = "Col de Teghime"; name = "Col de Teghime"; massif = "Corse" },
    @{ source = "Col de Sevi"; name = "Col de Sevi"; massif = "Corse" },
    @{ source = "Col de Bigorno"; name = "Col de Bigorno"; massif = "Corse" },
    @{ source = "Col de Saint Eustache"; name = "Col de Saint-Eustache"; massif = "Corse" },
    @{ source = "Col de la Vaccia"; name = "Col de la Vaccia"; massif = "Corse" },
    @{ source = "Col de Palmarella"; name = "Col de Palmarella"; massif = "Corse" },

    @{ source = "Plateau de Beille"; name = "Plateau de Beille"; massif = "Pyrénées" },
    @{ source = "Station de Luz Ardiden"; name = "Luz Ardiden"; massif = "Pyrénées" },
    @{ source = "Station de Superbagneres"; name = "Superbagnères"; massif = "Pyrénées" },
    @{ source = "Plateau d'Ax Bonascre"; name = "Ax 3 Domaines / Bonascre"; massif = "Pyrénées" },
    @{ source = "Col du Pourtalet"; name = "Col du Pourtalet"; massif = "Pyrénées" },
    @{ source = "Col de Tentes"; name = "Col de Tentes"; massif = "Pyrénées" },
    @{ source = "Col de Péguère"; name = "Col de Péguère"; massif = "Pyrénées" },
    @{ source = "Col Bagargui"; name = "Col Bagargui"; massif = "Pyrénées" },
    @{ source = "Station de Piau Engaly"; name = "Piau-Engaly"; massif = "Pyrénées" },
    @{ source = "Bout de Touron - Prat d'Albis"; name = "Prat d'Albis"; massif = "Pyrénées" },
    @{ source = "Pont d'Espagne"; name = "Pont d'Espagne"; massif = "Pyrénées" },
    @{ source = "Col de Latrape"; name = "Col de Latrape"; massif = "Pyrénées" },

    @{ source = "Col des Glières"; name = "Col des Glières"; massif = "Alpes" },
    @{ source = "Mont du Chat"; name = "Mont du Chat"; massif = "Jura" },
    @{ source = "Col du Granier"; name = "Col du Granier"; massif = "Alpes" },
    @{ source = "Col de la Croix Fry"; name = "Col de la Croix Fry"; massif = "Alpes" },
    @{ source = "Col de la Forclaz de Montmin"; name = "Col de la Forclaz de Montmin"; massif = "Alpes" },
    @{ source = "Col de la Machine"; name = "Col de la Machine"; massif = "Alpes" },
    @{ source = "Col du Coq"; name = "Col du Coq"; massif = "Alpes" },
    @{ source = "Col de Turini"; name = "Col de Turini"; massif = "Alpes" },
    @{ source = "Signal de Lure"; name = "Signal de Lure"; massif = "Alpes" },
    @{ source = "Mont Faron"; name = "Mont Faron"; massif = "Alpes" },
    @{ source = "Roche Beranger (Station de Chamrousse)"; name = "Chamrousse (Roche Béranger)"; massif = "Alpes" },
    @{ source = "Col du Mont Cenis"; name = "Col du Mont-Cenis"; massif = "Alpes" },

    @{ source = "Mont Aigoual"; name = "Mont Aigoual"; massif = "Massif central" },
    @{ source = "Col de la Lusette"; name = "Col de la Lusette"; massif = "Massif central" },
    @{ source = "Col du Béal"; name = "Col du Béal"; massif = "Massif central" },
    @{ source = "Col de Néronne"; name = "Col de Néronne"; massif = "Massif central" },
    @{ source = "Col de la Croix de Chaubouret"; name = "Col de la Croix de Chaubouret"; massif = "Massif central" },
    @{ source = "Col de la République / Col de Grand Bois"; name = "Col de la République / Grand Bois"; massif = "Massif central" },
    @{ source = "Col du Gerbier de Jonc"; name = "Col du Gerbier-de-Jonc"; massif = "Massif central" },

    @{ source = "Col de Berthiand"; name = "Col de Berthiand"; massif = "Jura" },
    @{ source = "Col de la Savine"; name = "Col de la Savine"; massif = "Jura" },
    @{ source = "Le Mont d'Or"; name = "Mont d'Or"; massif = "Jura" },

    @{ source = "Station de La Plagne 2000"; name = "La Plagne 2000"; massif = "Alpes" },
    @{ source = "Station de La Toussuire"; name = "La Toussuire"; massif = "Alpes" },
    @{ source = "Station de Tignes"; name = "Tignes"; massif = "Alpes" },
    @{ source = "Station de Pra Loup"; name = "Pra-Loup"; massif = "Alpes" },
    @{ source = "Station des Orres"; name = "Les Orres"; massif = "Alpes" },
    @{ source = "Station de Puy-Saint-Vincent"; name = "Puy-Saint-Vincent"; massif = "Alpes" },
    @{ source = "Station de Cauterets-Campbasque"; name = "Cauterets-Cambasque"; massif = "Pyrénées" },
    @{ source = "Cirque de Troumouse"; name = "Cirque de Troumouse"; massif = "Pyrénées" },
    @{ source = "Lac de Cap de Long"; name = "Lac de Cap-de-Long"; massif = "Pyrénées" }
)

$profileManifest = @'
Ballon d'Alsace|/vosges/france/ballon-d-alsace-depuis-malvaux-c24.htm
Ballon d'Alsace|/vosges/france/ballon-d-alsace-depuis-saint-maurice-sur-moselle-c26.htm
Ballon d'Alsace|/vosges/france/ballon-d-alsace-depuis-sewen-c25.htm
Grand Ballon|/vosges/france/grand-ballon-depuis-cernay-c79.htm
Grand Ballon|/vosges/france/grand-ballon-depuis-moosch-c1440.htm
Grand Ballon|/vosges/france/grand-ballon-depuis-saint-amarin-c78.htm
Grand Ballon|/vosges/france/grand-ballon-depuis-soultz-c80.htm
Grand Ballon|/vosges/france/grand-ballon-depuis-wattwiller-c2115.htm
Grand Ballon|/vosges/france/grand-ballon-depuis-willer-sur-thur-c81.htm
Petit Ballon|/vosges/france/petit-ballon-depuis-metzeral-c389.htm
Petit Ballon|/vosges/france/petit-ballon-depuis-munster-c387.htm
Petit Ballon|/vosges/france/petit-ballon-depuis-wihr-au-val-c388.htm
Col de Platzerwasel|/vosges/france/col-de-platzerwasel-depuis-munster-c1158.htm
Station de la Planche des Belles Filles|/vosges/france/station-de-la-planche-des-belles-filles-depuis-plancher-les-mines-c1126.htm
Col de la Schlucht|/vosges/france/col-de-la-schlucht-depuis-fraize-c230.htm
Col de la Schlucht|/vosges/france/col-de-la-schlucht-depuis-la-bresse-c229.htm
Col de la Schlucht|/vosges/france/col-de-la-schlucht-depuis-le-kertoff-c227.htm
Col de la Schlucht|/vosges/france/col-de-la-schlucht-depuis-munster-c228.htm
Col de la Schlucht|/vosges/france/col-de-la-schlucht-depuis-xonrupt-c1228.htm
Station de Champ du feu|/vosges/france/station-de-champ-du-feu-depuis-fouday-c378.htm
Station de Champ du feu|/vosges/france/station-de-champ-du-feu-depuis-obernai-c381.htm
Station de Champ du feu|/vosges/france/station-de-champ-du-feu-depuis-saint-pierre-c382.htm
Station de Champ du feu|/vosges/france/station-de-champ-du-feu-depuis-schirmeck-c379.htm
Station de Champ du feu|/vosges/france/station-de-champ-du-feu-depuis-ville-c380.htm
Col du Vergio|/corse/france/col-du-vergio-depuis-porto-marina-c2681.htm
Col de Bavella|/corse/france/col-de-bavella-depuis-solenzara-c2682.htm
Col de Bavella|/corse/france/col-de-bavella-depuis-zonza-c664.htm
Col de Vizzavona|/corse/france/col-de-vizzavona-depuis-n193-d29-c1782.htm
Col de Vizzavona|/corse/france/col-de-vizzavona-depuis-vivario-c1780.htm
Col de l'Ospedale|/corse/france/col-de-l-ospedale-depuis-palavese-c2698.htm
Col de Scalella|/corse/france/col-de-scalella-depuis-bastelica-c2161.htm
Col de Scalella|/corse/france/col-de-scalella-depuis-tavera-c2685.htm
Gorges de la Restonica|/corse/france/gorges-de-la-restonica-depuis-corte-c2688.htm
Col de Verde|/corse/france/col-de-verde-depuis-pont-de-cannareccia-c3564.htm|Pont de Cannareccia
Col de Sorba|/corse/france/col-de-sorba-depuis-ghisoni-c2690.htm
Col de Sorba|/corse/france/col-de-sorba-depuis-vivario-c2689.htm
Col de Prato|/corse/france/col-de-prato-depuis-pont-de-rimitorio-c2716.htm
Col de Prato|/corse/france/col-de-prato-depuis-ponte-leccia-c2171.htm
Col de Teghime|/corse/france/col-de-teghime-depuis-bastia-c1681.htm
Col de Teghime|/corse/france/col-de-teghime-depuis-montesoro-c1684.htm
Col de Teghime|/corse/france/col-de-teghime-depuis-saint-florent-c1682.htm|Saint-Florent, via Patrimonio|checkpoint
Col de Teghime|/corse/france/col-de-teghime-depuis-saint-florent-c1683.htm|Saint-Florent, via Oletta|checkpoint
Col de Sevi|/corse/france/col-de-sevi-depuis-d84-d124-c2684.htm
Col de Sevi|/corse/france/col-de-sevi-depuis-sagone-c2683.htm
Col de Bigorno|/corse/france/col-de-bigorno-depuis-biguglia-c2559.htm
Col de Bigorno|/corse/france/col-de-bigorno-depuis-ponte-novu-c2713.htm
Col de Bigorno|/corse/france/col-de-bigorno-depuis-volpajola-c2170.htm
Col de Saint Eustache|/corse/france/col-de-saint-eustache-depuis-moca-c2699.htm
Col de Saint Eustache|/corse/france/col-de-saint-eustache-depuis-moulin-d-arnia-c3566.htm
Col de Saint Eustache|/corse/france/col-de-saint-eustache-depuis-petreto-c2700.htm
Col de la Vaccia|/corse/france/col-de-la-vaccia-depuis-olivese-c2704.htm
Col de Palmarella|/corse/france/col-de-palmarella-depuis-le-fango-c1789.htm
Col de Palmarella|/corse/france/col-de-palmarella-depuis-porto-c1790.htm
Plateau de Beille|/pyrenees-centrales/france/plateau-de-beille-depuis-les-cabannes-c27.htm
Station de Luz Ardiden|/pyrenees-centrales/france/station-de-luz-ardiden-depuis-luz-saint-sauveur-c61.htm
Station de Luz Ardiden|/pyrenees-centrales/france/station-de-luz-ardiden-depuis-viscos-c1508.htm
Station de Superbagneres|/pyrenees-centrales/france/station-de-superbagneres-depuis-bagneres-de-luchon-c538.htm
Plateau d'Ax Bonascre|/pyrenees-centrales/france/plateau-d-ax-bonascre-depuis-ax-les-thermes-c14.htm
Col du Pourtalet|/pyrenees-ouest/france/col-du-pourtalet-depuis-laruns-c539.htm
Col du Pourtalet|/pyrenees-ouest/espagne/col-du-pourtalet-depuis-biescas-via-hoz-de-jaca-c2602.htm
Col du Pourtalet|/pyrenees-ouest/espagne/col-du-pourtalet-depuis-bubal-c252.htm
Col de Tentes|/pyrenees-centrales/france/col-de-tentes-depuis-luz-saint-sauveur-c1859.htm
Col de Péguère|/pyrenees-centrales/france/col-de-peguere-depuis-estaniels-c1638.htm
Col de Péguère|/pyrenees-centrales/france/col-de-peguere-depuis-la-mouline-c1368.htm
Col de Péguère|/pyrenees-centrales/france/col-de-peguere-depuis-massat-c1637.htm
Col Bagargui|/pyrenees-ouest/france/col-bagargui-depuis-alcay-c1844.htm
Col Bagargui|/pyrenees-ouest/france/col-bagargui-depuis-est-c23.htm
Col Bagargui|/pyrenees-ouest/france/col-bagargui-depuis-esterencuby-c22.htm
Col Bagargui|/pyrenees-ouest/france/col-bagargui-depuis-saint-jean-le-vieux-c616.htm
Station de Piau Engaly|/pyrenees-centrales/france/station-de-piau-engaly-depuis-saint-lary-soulan-c312.htm
Bout de Touron - Prat d'Albis|/pyrenees-centrales/france/bout-de-touron-prat-d-albis-depuis-foix-c1366.htm
Pont d'Espagne|/pyrenees-centrales/france/pont-d-espagne-depuis-pierrefitte-nestalas-c1195.htm
Col de Latrape|/pyrenees-centrales/france/col-de-latrape-depuis-aulus-les-bains-c1636.htm
Col de Latrape|/pyrenees-centrales/france/col-de-latrape-depuis-serac-c1635.htm
Col des Glières|/aravis/france/col-des-glieres-depuis-le-petit-bornand-c1652.htm
Col des Glières|/aravis/france/col-des-glieres-depuis-nant-sec-thorens-glieres-c1653.htm
Mont du Chat|/bugey/france/mont-du-chat-depuis-d921-d41a-c2454.htm
Mont du Chat|/bugey/france/mont-du-chat-depuis-le-bourget-du-lac-c142.htm
Mont du Chat|/bugey/france/mont-du-chat-depuis-yenne-c143.htm
Col du Granier|/chartreuse/france/col-du-granier-depuis-chambery-c108.htm
Col du Granier|/chartreuse/france/col-du-granier-depuis-chapareillan-c106.htm
Col du Granier|/chartreuse/france/col-du-granier-depuis-saint-baldoph-c109.htm
Col du Granier|/chartreuse/france/col-du-granier-depuis-saint-pierre-d-entremont-c107.htm
Col de la Croix Fry|/aravis/france/col-de-la-croix-fry-depuis-saint-jean-de-sixt-c504.htm
Col de la Croix Fry|/aravis/france/col-de-la-croix-fry-depuis-thones-c505.htm
Col de la Forclaz de Montmin|/aravis/france/col-de-la-forclaz-de-montmin-depuis-menthon-saint-bernard-c43.htm
Col de la Forclaz de Montmin|/aravis/france/col-de-la-forclaz-de-montmin-depuis-vesonne-c42.htm
Col de la Machine|/vercors/france/col-de-la-machine-depuis-saint-jean-en-royans-c570.htm
Col de la Machine|/vercors/france/col-de-la-machine-depuis-saint-laurent-en-royans-c571.htm
Col du Coq|/chartreuse/france/col-du-coq-depuis-saint-hugues-c914.htm
Col du Coq|/chartreuse/france/col-du-coq-depuis-saint-nazaire-les-eymes-c913.htm
Col de Turini|/prealpes-de-nice/france/col-de-turini-depuis-d-2565-c140.htm
Col de Turini|/prealpes-de-nice/france/col-de-turini-depuis-l-escarene-c3864.htm
Col de Turini|/prealpes-de-nice/france/col-de-turini-depuis-sospel-c141.htm
Signal de Lure|/provence/france/signal-de-lure-depuis-saint-etienne-les-orgues-c313.htm
Signal de Lure|/provence/france/signal-de-lure-depuis-valbelle-vallee-du-jabron-c865.htm
Mont Faron|/monts-toulonnais/france/mont-faron-depuis-toulon-c176.htm
Roche Beranger (Station de Chamrousse)|/belledonne/france/roche-beranger-station-de-chamrousse-depuis-uriage-les-bains-c130.htm
Roche Beranger (Station de Chamrousse)|/belledonne/france/roche-beranger-station-de-chamrousse-depuis-vaulnaveys-le-haut-c1276.htm
Col du Mont Cenis|/massif-du-mont-cenis/france/col-du-mont-cenis-depuis-lanslebourg-c87.htm
Col du Mont Cenis|/massif-du-mont-cenis/italie/col-du-mont-cenis-depuis-susa-c88.htm
Mont Aigoual|/cevennes/france/mont-aigoual-depuis-le-vigan-c154.htm
Mont Aigoual|/cevennes/france/mont-aigoual-depuis-le-vigan-via-le-col-de-la-lusette-c2207.htm
Mont Aigoual|/cevennes/france/mont-aigoual-depuis-meyrueis-c152.htm
Mont Aigoual|/cevennes/france/mont-aigoual-depuis-rousses-c151.htm
Mont Aigoual|/cevennes/france/mont-aigoual-depuis-saint-andre-de-valborgne-c671.htm
Mont Aigoual|/cevennes/france/mont-aigoual-depuis-valleraugue-c153.htm
Col de la Lusette|/cevennes/france/col-de-la-lusette-depuis-l-arboux-c669.htm
Col de la Lusette|/cevennes/france/col-de-la-lusette-depuis-la-valette-c670.htm
Col de la Lusette|/cevennes/france/col-de-la-lusette-depuis-le-vigan-c2206.htm
Col du Béal|/livradois-forez/france/col-du-beal-depuis-job-c6.htm
Col du Béal|/livradois-forez/france/col-du-beal-depuis-le-brugeron-c974.htm
Col du Béal|/livradois-forez/france/col-du-beal-depuis-leigneux-c3.htm
Col du Béal|/livradois-forez/france/col-du-beal-depuis-vertolaye-c892.htm
Col de Néronne|/monts-du-cantal/france/col-de-neronne-depuis-salers-c345.htm
Col de la Croix de Chaubouret|/massif-du-pilat/france/col-de-la-croix-de-chaubouret-depuis-bourg-argental-c1397.htm
Col de la Croix de Chaubouret|/massif-du-pilat/france/col-de-la-croix-de-chaubouret-depuis-saint-chamond-c946.htm
Col de la Croix de Chaubouret|/massif-du-pilat/france/col-de-la-croix-de-chaubouret-depuis-saint-etienne-c500.htm
Col de la République / Col de Grand Bois|/massif-du-pilat/france/col-de-la-republique-col-de-grand-bois-depuis-bourg-argental-c967.htm
Col de la République / Col de Grand Bois|/massif-du-pilat/france/col-de-la-republique-col-de-grand-bois-depuis-saint-etienne-c501.htm
Col du Gerbier de Jonc|/monts-du-vivarais/france/col-du-gerbier-de-jonc-depuis-la-chazotte-c2372.htm
Col du Gerbier de Jonc|/monts-du-vivarais/france/col-du-gerbier-de-jonc-depuis-le-bourlatier-d122-d378-c720.htm
Col du Gerbier de Jonc|/monts-du-vivarais/france/col-du-gerbier-de-jonc-depuis-sainte-eulalie-c757.htm
Col de Berthiand|/jura/france/col-de-berthiand-depuis-nurieux-volognat-c1456.htm
Col de Berthiand|/jura/france/col-de-berthiand-depuis-serrieres-sur-ain-c665.htm
Col de la Savine|/jura/france/col-de-la-savine-depuis-champagnole-c1756.htm
Col de la Savine|/jura/france/col-de-la-savine-depuis-morez-c1755.htm
Le Mont d'Or|/jura/france/le-mont-d-or-depuis-longevilles-mt-d-or-c1281.htm
Station de La Plagne 2000|/vanoise/france/station-de-la-plagne-2000-depuis-aime-c38.htm
Station de La Toussuire|/arves-et-grandes-rousses/france/station-de-la-toussuire-depuis-saint-jean-de-maurienne-c417.htm
Station de La Toussuire|/arves-et-grandes-rousses/france/station-de-la-toussuire-depuis-saint-jean-de-maurienne-via-saint-pancrace-c416.htm
Station de Tignes|/vanoise/france/station-de-tignes-depuis-sainte-foy-tarentaise-c785.htm
Station de Pra Loup|/trois-eveches/france/station-de-pra-loup-depuis-uvernet-fours-d908-d109-c211.htm
Station des Orres|/ubaye/france/station-des-orres-depuis-embrun-c186.htm
Station de Puy-Saint-Vincent|/ecrins/france/station-de-puy-saint-vincent-depuis-l-argentiere-la-bessee-c1530.htm
Station de Cauterets-Campbasque|/pyrenees-centrales/france/station-de-cauterets-campbasque-depuis-pierrefitte-nestalas-c711.htm
Cirque de Troumouse|/pyrenees-centrales/france/cirque-de-troumouse-depuis-luz-saint-sauveur-c1459.htm
Lac de Cap de Long|/pyrenees-centrales/france/lac-de-cap-de-long-depuis-saint-lary-soulan-c311.htm
'@

$profileRows = [System.Collections.Generic.List[object]]::new()
foreach ($line in ($profileManifest -split "`n")) {
    $trimmedLine = $line.Trim()
    if (-not $trimmedLine) { continue }
    $parts = $trimmedLine.Split("|")
    $profileRows.Add([pscustomobject]@{
        sourceName = $parts[0]
        url = "https://www.cols-cyclisme.com$($parts[1])"
        alternativeName = if ($parts.Count -ge 3) { $parts[2] } else { "" }
        checkpoint = $parts.Count -ge 4 -and $parts[3] -eq "checkpoint"
    })
}

$climbs = [System.Collections.Generic.List[object]]::new()
$profileCount = $profileRows.Count
$processed = 0
foreach ($definition in $definitions) {
    $matchingRows = @($profileRows | Where-Object { $_.sourceName -eq $definition.source })
    if ($matchingRows.Count -eq 0) {
        throw "No profile found for '$($definition.source)'"
    }

    $profiles = [System.Collections.Generic.List[object]]::new()
    foreach ($profileRow in $matchingRows) {
        $profile = Get-ProfileData $profileRow.url $profileRow.alternativeName
        if ($profileRow.checkpoint) {
            $profile.alternative | Add-Member `
                -NotePropertyName routeCheckpoints `
                -NotePropertyValue @($profile.midpointCoordinate) `
                -Force
        }
        $profiles.Add($profile)
        $processed++
        if (($processed % 10) -eq 0 -or $processed -eq $profileCount) {
            Write-Host "Profiles: $processed / $profileCount"
        }
    }

    $firstProfile = $profiles[0]
    $climbs.Add([pscustomobject][ordered]@{
        name = $definition.name
        country = "FR"
        massif = $definition.massif
        topOfTheAscent = $firstProfile.topAltitude
        geoCoordinate = $firstProfile.summitCoordinate
        alternatives = @($profiles | ForEach-Object { $_.alternative })
    })
}

$madeleine = $climbs | Where-Object { $_.name -eq "Col de la Madeleine" } | Select-Object -First 1
$madeleineDirect = $madeleine.alternatives | Where-Object { $_.name -eq "La Chambre" } | Select-Object -First 1
if ($madeleineDirect) {
    $madeleineDirect.name = "La Chambre, par la D213"
    $madeleineDirect | Add-Member -NotePropertyName routeCheckpoints -NotePropertyValue @(
        [pscustomobject][ordered]@{ latitude = 45.386825; longitude = 6.331231 }
    ) -Force
}

# The published Sainte-Eulalie profile describes only the final 2.5 km, while
# its GPX start point is in Sainte-Eulalie. Keep the full village-to-summit
# distance so badge matching and the displayed average remain consistent.
$gerbier = $climbs | Where-Object { $_.name -eq "Col du Gerbier-de-Jonc" } | Select-Object -First 1
$gerbierSainteEulalie = $gerbier.alternatives | Where-Object { $_.name -eq "Sainte-Eulalie" } | Select-Object -First 1
if ($gerbierSainteEulalie) {
    $gerbierSainteEulalie.length = 5.0
    $gerbierSainteEulalie.difficulty = 63
    $gerbierSainteEulalie.averageGradient = 3.54
}

$variantDefinitions = @(
    @{
        climbName = "Alpe d'Huez"
        name = "Rochetaille via Villard-Reculas"
        url = "https://www.cols-cyclisme.com/arves-et-grandes-rousses/france/station-de-l-alpe-d-huez-depuis-rochetaille-c4.htm"
    },
    @{
        climbName = "Col de la Madeleine"
        name = "La Chambre, via Montgellafrey"
        url = "https://www.cols-cyclisme.com/vanoise/france/col-de-la-madeleine-depuis-la-chambre-via-montgellafrey-c3112.htm"
        routeCheckpoints = @(
            [pscustomobject][ordered]@{ latitude = 45.391775; longitude = 6.319134 }
        )
    },
    @{
        climbName = "Col de Soudet"
        name = "D918 / D632 via la Hourcère"
        url = "https://www.cols-cyclisme.com/pyrenees-ouest/france/col-de-soudet-depuis-d918-d632-c332.htm"
    }
)

$variants = [System.Collections.Generic.List[object]]::new()
foreach ($variantDefinition in $variantDefinitions) {
    $profile = Get-ProfileData $variantDefinition.url $variantDefinition.name
    if ($variantDefinition.routeCheckpoints) {
        $profile.alternative | Add-Member `
            -NotePropertyName routeCheckpoints `
            -NotePropertyValue @($variantDefinition.routeCheckpoints) `
            -Force
    }
    $variants.Add([pscustomobject][ordered]@{
        climbName = $variantDefinition.climbName
        alternative = $profile.alternative
    })
}

$result = [pscustomobject][ordered]@{
    generatedAt = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
    source = "https://www.cols-cyclisme.com/france/liste-p1.htm"
    climbs = @($climbs)
    variants = @($variants)
}

$json = ($result | ConvertTo-Json -Depth 10) -replace "`r`n", "`n"
$resolvedOutputPath = [System.IO.Path]::GetFullPath($OutputPath)
$outputDirectory = [System.IO.Path]::GetDirectoryName($resolvedOutputPath)
if ($outputDirectory -and -not [System.IO.Directory]::Exists($outputDirectory)) {
    [System.IO.Directory]::CreateDirectory($outputDirectory) | Out-Null
}
[System.IO.File]::WriteAllText(
    $resolvedOutputPath,
    "$json`n",
    [System.Text.UTF8Encoding]::new($false)
)
Write-Host "Generated $($climbs.Count) climbs and $($variants.Count) variants in $resolvedOutputPath"
