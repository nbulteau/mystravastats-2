param(
    [Parameter(Mandatory = $true)]
    [string] $OutputPath,

    [string[]] $ExistingCatalogPaths = @()
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"
$invariantCulture = [System.Globalization.CultureInfo]::InvariantCulture
$baseUrl = "https://www.cols-cyclisme.com"
$catalogUrl = "$baseUrl/espagne/liste-p2.htm"
$cacheDirectory = Join-Path ([System.IO.Path]::GetTempPath()) "mystravastats-cols-cyclisme-cache"
[System.IO.Directory]::CreateDirectory($cacheDirectory) | Out-Null
$excludedProfileIds = @{
    # The source lists 2.22 km at 7.34% with a 6.3% maximum, while its GPX
    # endpoints are already 4.03 km apart. Exclude instead of inventing a route.
    "1073" = "inconsistent length, maximum gradient and GPX geometry"
}

function Invoke-CatalogRequest {
    param([Parameter(Mandatory = $true)][string] $Url)

    $urlBytes = [System.Text.Encoding]::UTF8.GetBytes($Url)
    $cacheKey = [Convert]::ToHexString([System.Security.Cryptography.SHA256]::HashData($urlBytes))
    $cachePath = Join-Path $cacheDirectory "$cacheKey.txt"
    if ([System.IO.File]::Exists($cachePath)) {
        return [System.IO.File]::ReadAllText($cachePath)
    }

    Start-Sleep -Milliseconds 500
    for ($attempt = 1; $attempt -le 10; $attempt++) {
        try {
            $response = Invoke-WebRequest `
                -UseBasicParsing `
                -Uri $Url `
                -TimeoutSec 60 `
                -Headers @{ "User-Agent" = "Mozilla/5.0 (compatible; MyStravaStats famous-climb catalog refresh)" }
            $joinedContent = $response.Content
        }
        catch {
            $joinedContent = ""
        }
        if ($joinedContent) {
            [System.IO.File]::WriteAllText($cachePath, $joinedContent, [System.Text.UTF8Encoding]::new($false))
            return $joinedContent
        }
        if ($attempt -eq 10) {
            throw "Unable to download $Url after $attempt attempts"
        }
        Start-Sleep -Seconds ([math]::Min(4 * $attempt, 20))
    }
}

function ConvertFrom-HtmlCell {
    param([string] $Value)

    $withoutTags = $Value -replace "<[^>]+>", ""
    return [System.Net.WebUtility]::HtmlDecode(($withoutTags -replace "\s+", " ")).Trim()
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
        if ($Optional) { return 0.0 }
        throw "Metric '$Label' not found"
    }

    $text = ConvertFrom-HtmlCell $match.Groups["value"].Value
    $number = [regex]::Match($text, "-?\d+(?:[\.,]\d+)?")
    if (-not $number.Success) {
        if ($Optional) { return 0.0 }
        throw "Metric '$Label' has no numeric value: $text"
    }

    return [double]::Parse($number.Value.Replace(",", "."), $invariantCulture)
}

function Get-ClimbCategory {
    param([int] $Difficulty)

    if ($Difficulty -ge 1000) { return "HC" }
    if ($Difficulty -ge 600) { return "1" }
    if ($Difficulty -ge 300) { return "2" }
    if ($Difficulty -ge 150) { return "3" }
    return "4"
}

function Get-HaversineDistanceKm {
    param(
        [double] $Latitude1,
        [double] $Longitude1,
        [double] $Latitude2,
        [double] $Longitude2
    )

    $earthRadiusKm = 6371.0
    $radians = [math]::PI / 180.0
    $latitudeDelta = ($Latitude2 - $Latitude1) * $radians
    $longitudeDelta = ($Longitude2 - $Longitude1) * $radians
    $value = [math]::Sin($latitudeDelta / 2) * [math]::Sin($latitudeDelta / 2) +
        [math]::Cos($Latitude1 * $radians) * [math]::Cos($Latitude2 * $radians) *
        [math]::Sin($longitudeDelta / 2) * [math]::Sin($longitudeDelta / 2)
    return 2 * $earthRadiusKm * [math]::Atan2([math]::Sqrt($value), [math]::Sqrt(1 - $value))
}

function Get-ExistingLabels {
    param([string[]] $Paths)

    $labels = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)
    foreach ($path in $Paths) {
        if (-not [System.IO.File]::Exists($path)) {
            throw "Existing catalog not found: $path"
        }
        $catalog = Get-Content -Raw $path | ConvertFrom-Json
        foreach ($climb in $catalog) {
            foreach ($alternative in $climb.alternatives) {
                [void] $labels.Add("$($climb.name) from $($alternative.name)")
            }
        }
    }
    return $labels
}

function Get-CatalogRows {
    param([Parameter(Mandatory = $true)][string] $Html)

    $rows = [System.Collections.Generic.List[object]]::new()
    $rowMatches = [regex]::Matches(
        $Html,
        '<tr[^>]*onclick="location\.href=''(?<url>[^'']+-c\d+\.htm)''"[^>]*>(?<cells>.*?)</tr>',
        [System.Text.RegularExpressions.RegexOptions]::IgnoreCase -bor
            [System.Text.RegularExpressions.RegexOptions]::Singleline
    )
    foreach ($rowMatch in $rowMatches) {
        $cells = [regex]::Matches(
            $rowMatch.Groups["cells"].Value,
            "<td[^>]*>(?<value>.*?)</td>",
            [System.Text.RegularExpressions.RegexOptions]::IgnoreCase -bor
                [System.Text.RegularExpressions.RegexOptions]::Singleline
        )
        if ($cells.Count -lt 5) { continue }

        $country = ConvertFrom-HtmlCell $cells[4].Groups["value"].Value
        if ($country -ne "Espagne") { continue }

        $relativeUrl = $rowMatch.Groups["url"].Value
        $rows.Add([pscustomobject][ordered]@{
            name = ConvertFrom-HtmlCell $cells[0].Groups["value"].Value
            startName = ConvertFrom-HtmlCell $cells[1].Groups["value"].Value
            listedAltitude = [int][math]::Round([double]::Parse(
                ([regex]::Match((ConvertFrom-HtmlCell $cells[2].Groups["value"].Value), "\d+").Value),
                $invariantCulture
            ))
            massif = ConvertFrom-HtmlCell $cells[3].Groups["value"].Value
            sourceUrl = if ($relativeUrl.StartsWith("http")) { $relativeUrl } else { "$baseUrl$relativeUrl" }
        })
    }

    return @($rows | Sort-Object sourceUrl -Unique)
}

function Get-ProfileData {
    param([Parameter(Mandatory = $true)] $Row)

    $html = Invoke-CatalogRequest $Row.sourceUrl
    $profileIdMatch = [regex]::Match($Row.sourceUrl, "-c(?<id>\d+)\.htm$")
    if (-not $profileIdMatch.Success) {
        throw "Unable to determine profile id from $($Row.sourceUrl)"
    }
    $profileId = $profileIdMatch.Groups["id"].Value

    $length = Get-TableMetric $html "Longueur :"
    $totalAscent = [int][math]::Round((Get-TableMetric $html "Dénivellation :"))
    $averageGradient = Get-TableMetric $html "% Moyen :"
    $maximumGradient = Get-TableMetric $html "% Maximal :" -Optional
    $topAltitude = [int][math]::Round((Get-TableMetric $html "Altitude :"))

    [xml] $gpx = Invoke-CatalogRequest "$baseUrl/gpx/$profileId.gpx"
    $trackPoints = @($gpx.SelectNodes("//*[local-name()='trkpt']"))
    if ($trackPoints.Count -lt 2) {
        throw "GPX $profileId does not contain enough track points"
    }

    $first = $trackPoints[0]
    $last = $trackPoints[-1]
    $firstElevation = [double]::Parse($first.ele, $invariantCulture)
    $lastElevation = [double]::Parse($last.ele, $invariantCulture)
    $summitIsFirst = [math]::Abs($firstElevation - $topAltitude) -le [math]::Abs($lastElevation - $topAltitude)
    if ($summitIsFirst) {
        $summit = $first
    }
    else {
        $summit = $last
    }

    $profilePoints = @($trackPoints)
    if (-not $summitIsFirst) {
        [array]::Reverse($profilePoints)
    }
    $startIndex = $profilePoints.Count - 1
    $directDistance = Get-HaversineDistanceKm `
        -Latitude1 ([double]::Parse($profilePoints[0].lat, $invariantCulture)) `
        -Longitude1 ([double]::Parse($profilePoints[0].lon, $invariantCulture)) `
        -Latitude2 ([double]::Parse($profilePoints[-1].lat, $invariantCulture)) `
        -Longitude2 ([double]::Parse($profilePoints[-1].lon, $invariantCulture))
    if ($directDistance -gt $length + 0.5) {
        $coveredDistance = 0.0
        for ($pointIndex = 1; $pointIndex -lt $profilePoints.Count; $pointIndex++) {
            $previous = $profilePoints[$pointIndex - 1]
            $current = $profilePoints[$pointIndex]
            $coveredDistance += Get-HaversineDistanceKm `
                -Latitude1 ([double]::Parse($previous.lat, $invariantCulture)) `
                -Longitude1 ([double]::Parse($previous.lon, $invariantCulture)) `
                -Latitude2 ([double]::Parse($current.lat, $invariantCulture)) `
                -Longitude2 ([double]::Parse($current.lon, $invariantCulture))
            if ($coveredDistance -ge $length) {
                $startIndex = $pointIndex
                break
            }
        }
    }
    $start = $profilePoints[$startIndex]
    $selectedProfilePoints = @($profilePoints[0..$startIndex])
    $minimumAltitude = [int][math]::Round((
        $selectedProfilePoints |
            ForEach-Object { [double]::Parse($_.ele, $invariantCulture) } |
            Measure-Object -Minimum
    ).Minimum)
    $difficulty = [int][math]::Round($length * $averageGradient * $averageGradient)

    return [pscustomobject][ordered]@{
        name = $Row.name
        massif = $Row.massif
        topAltitude = $topAltitude
        summitCoordinate = [pscustomobject][ordered]@{
            latitude = [math]::Round([double]::Parse($summit.lat, $invariantCulture), 6)
            longitude = [math]::Round([double]::Parse($summit.lon, $invariantCulture), 6)
        }
        alternative = [pscustomobject][ordered]@{
            name = $Row.startName
            geoCoordinate = [pscustomobject][ordered]@{
                latitude = [math]::Round([double]::Parse($start.lat, $invariantCulture), 6)
                longitude = [math]::Round([double]::Parse($start.lon, $invariantCulture), 6)
            }
            length = [math]::Round($length, 2)
            totalAscent = $totalAscent
            minimumAltitude = $minimumAltitude
            maximumGradient = [math]::Round($maximumGradient, 1)
            difficulty = $difficulty
            averageGradient = [math]::Round($averageGradient, 2)
            category = Get-ClimbCategory $difficulty
            sourceUrl = $Row.sourceUrl
        }
    }
}

$existingLabels = Get-ExistingLabels $ExistingCatalogPaths
$catalogRows = Get-CatalogRows (Invoke-CatalogRequest $catalogUrl)
if ($catalogRows.Count -eq 0) {
    throw "No Spanish climb profiles found in $catalogUrl"
}

$profiles = [System.Collections.Generic.List[object]]::new()
$skippedLabels = [System.Collections.Generic.List[string]]::new()
$excludedLabels = [System.Collections.Generic.List[string]]::new()
$index = 0
foreach ($row in $catalogRows) {
    $index++
    $label = "$($row.name) from $($row.startName)"
    $profileId = [regex]::Match($row.sourceUrl, "-c(?<id>\d+)\.htm$").Groups["id"].Value
    if ($excludedProfileIds.ContainsKey($profileId)) {
        $excludedLabels.Add("$label — $($excludedProfileIds[$profileId])")
        continue
    }
    if ($existingLabels.Contains($label)) {
        $skippedLabels.Add($label)
        continue
    }
    $profiles.Add((Get-ProfileData $row))
    if ($index % 10 -eq 0 -or $index -eq $catalogRows.Count) {
        Write-Host "Loaded $index / $($catalogRows.Count) Spanish profiles"
    }
}

$catalog = @(
    $profiles |
        Group-Object name |
        ForEach-Object {
            $group = @($_.Group | Sort-Object { $_.alternative.name })
            $first = $group[0]
            [pscustomobject][ordered]@{
                name = $first.name
                country = "ES"
                massif = $first.massif
                topOfTheAscent = $first.topAltitude
                geoCoordinate = $first.summitCoordinate
                alternatives = @($group | ForEach-Object { $_.alternative })
            }
        } |
        Sort-Object name
)

$outputDirectory = Split-Path -Parent $OutputPath
if ($outputDirectory) {
    [System.IO.Directory]::CreateDirectory($outputDirectory) | Out-Null
}
$json = $catalog | ConvertTo-Json -Depth 10
[System.IO.File]::WriteAllText($OutputPath, "$json`n", [System.Text.UTF8Encoding]::new($false))

$alternativeCount = ($catalog | ForEach-Object { $_.alternatives.Count } | Measure-Object -Sum).Sum
Write-Host "Wrote $($catalog.Count) Spanish summits / $alternativeCount sides to $OutputPath"
if ($skippedLabels.Count -gt 0) {
    Write-Host "Skipped $($skippedLabels.Count) labels already present in another national catalog:"
    $skippedLabels | Sort-Object | ForEach-Object { Write-Host "- $_" }
}
if ($excludedLabels.Count -gt 0) {
    Write-Host "Excluded $($excludedLabels.Count) profiles with inconsistent source data:"
    $excludedLabels | Sort-Object | ForEach-Object { Write-Host "- $_" }
}
