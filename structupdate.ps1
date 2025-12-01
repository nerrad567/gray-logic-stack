# structupdate.ps1
# Run from repo root: C:\Users\darre\OneDrive\vscode\gray-logic-stack

Write-Host "=== Gray Logic Stack – structure & placeholder setup ===`n"

# 1. Ensure directory structure exists
$dirs = @(
    "code",
    "code\stack",
    "code\scripts",
    "code\openhab",
    "code\node-red",
    "docs",
    "docs\diagrams",
    "docs\diagrams\src",
    "docs\diagrams\export",
    "docs\modules",
    "notes",
    "notes\meeting-notes"
)

foreach ($d in $dirs) {
    if (-not (Test-Path $d)) {
        Write-Host "Creating directory: $d"
        New-Item -ItemType Directory -Path $d | Out-Null
    }
}

# 2. Move docker-compose.yml into code\stack if it exists in code\
$oldCompose = "code\docker-compose.yml"
$newCompose = "code\stack\docker-compose.yml"
if (Test-Path $oldCompose) {
    Write-Host "Moving $oldCompose -> $newCompose"
    Move-Item $oldCompose $newCompose -Force
} elseif (Test-Path $newCompose) {
    Write-Host "docker-compose.yml already in code\stack (OK)"
} else {
    Write-Host "No docker-compose.yml found in code\ or code\stack (skipping)"
}

# 3. .env.example placeholder for the stack
$envExample = "code\stack\.env.example"
if (-not (Test-Path $envExample)) {
    Write-Host "Creating placeholder: $envExample"
    @(
        "# Gray Logic stack environment variables",
        "# Copy this file to .env and adjust values per site",
        "",
        "GL_SITE_NAME=example-site",
        "GL_ENVIRONMENT=lab",
        "GL_TIMEZONE=Europe/London"
    ) | Set-Content $envExample -Encoding UTF8
}

# 4. README placeholders for openhab/ and node-red/
$openhabReadme = "code\openhab\README.md"
if (-not (Test-Path $openhabReadme)) {
    Write-Host "Creating placeholder: $openhabReadme"
    @(
        "# openHAB configuration",
        "",
        "This directory will contain Gray Logic-specific openHAB configuration,",
        "such as things, items, rules, and UI definitions.",
        "",
        "Structure and usage:",
        "- `conf/` style layout to mirror an openHAB installation,",
        "- Example configs and templates for Gray Logic deployments."
    ) | Set-Content $openhabReadme -Encoding UTF8
}

$nodeRedReadme = "code\node-red\README.md"
if (-not (Test-Path $nodeRedReadme)) {
    Write-Host "Creating placeholder: $nodeRedReadme"
    @(
        "# Node-RED flows",
        "",
        "This directory will contain Node-RED flows and related assets",
        "used by the Gray Logic stack for cross-system logic and integrations.",
        "",
        "Typical contents:",
        "- Exported flow JSON files",
        "- Subflow definitions",
        "- Notes on how flows map to Gray Logic modules."
    ) | Set-Content $nodeRedReadme -Encoding UTF8
}

# 5. Scripts directory placeholders
$backupScript = "code\scripts\backup.sh"
if (-not (Test-Path $backupScript)) {
    Write-Host "Creating placeholder: $backupScript"
    @(
        "#!/usr/bin/env bash",
        "# Placeholder for Gray Logic backup script",
        "# TODO: implement backup of openHAB, Node-RED, and related volumes."
    ) | Set-Content $backupScript -Encoding UTF8
}

$restoreScript = "code\scripts\restore.sh"
if (-not (Test-Path $restoreScript)) {
    Write-Host "Creating placeholder: $restoreScript"
    @(
        "#!/usr/bin/env bash",
        "# Placeholder for Gray Logic restore script",
        "# TODO: implement restore procedure for Gray Logic deployments."
    ) | Set-Content $restoreScript -Encoding UTF8
}

# 6. Diagrams: split into src vs export

# Move existing PlantUML field-layer source into src
$oldFieldPuml = "docs\diagrams\field-layer.puml"
$newFieldPuml = "docs\diagrams\src\field-layer.puml"
if (Test-Path $oldFieldPuml) {
    Write-Host "Moving $oldFieldPuml -> $newFieldPuml"
    Move-Item $oldFieldPuml $newFieldPuml -Force
}

# Move existing network-segmentation.svg to export
$oldNetSvg = "docs\diagrams\network-segmentation.svg"
$newNetSvg = "docs\diagrams\export\network-segmentation.svg"
if (Test-Path $oldNetSvg) {
    Write-Host "Moving $oldNetSvg -> $newNetSvg"
    Move-Item $oldNetSvg $newNetSvg -Force
}

# Create PlantUML source placeholder if missing (correct quoting)
$netPuml = "docs\diagrams\src\network-segmentation.puml"
if (-not (Test-Path $netPuml)) {
    Write-Host "Creating placeholder: $netPuml"
    @(
        "@startuml",
        "' TODO: draw network segmentation diagram here",
        'title Network Segmentation - Gray Logic Stack',
        "",
        "' Example stencil:",
        'rectangle "Control LAN" { }',
        'rectangle "CCTV LAN" { }',
        'rectangle "Guest LAN" { }',
        '"Gray Logic Node" --> "VPS / Remote"',
        "",
        "@enduml"
    ) | Set-Content $netPuml -Encoding UTF8
}

# 7. Main spec file (single name, Git handles versioning)
# If an old gray-logic-stack-v0.2.md exists, rename it.
$oldSpecFile = "docs\gray-logic-stack-v0.2.md"
$specFile    = "docs\gray-logic-stack.md"

if ((Test-Path $oldSpecFile) -and -not (Test-Path $specFile)) {
    Write-Host "Renaming $oldSpecFile -> $specFile"
    Move-Item $oldSpecFile $specFile -Force
}

if (-not (Test-Path $specFile)) {
    Write-Host "Creating placeholder: $specFile"
    @(
        "# Gray Logic Stack – Working Draft",
        "",
        "> NOTE: Paste the current full spec text here.",
        "> Versioning will be tracked via Git tags/commits and CHANGELOG.md.",
        "",
        "## Status",
        "- Working draft",
        "- Owner: Darren Gray",
        "",
        "## Contents",
        "- What Gray Logic is",
        "- Why not Loxone / Crestron / Control4",
        "- Core goals and non-goals",
        "- Architecture overview (field, controller, infra layers)",
        "- Design rules (hard rules, strong rules, patterns)",
        "- Functional modules",
        "- Delivery model & roadmap",
        "",
        "_TODO: Replace this placeholder with the full spec._"
    ) | Set-Content $specFile -Encoding UTF8
} else {
    Write-Host "Spec file already exists: $specFile (leaving as-is)"
}

# 8. Module docs – stub missing ones
$modules = @(
    @{ Path = "docs\modules\core.md";           Title = "Core Module (Traefik, Dashboard, Metrics)" },
    @{ Path = "docs\modules\environment.md";    Title = "Environment Monitoring Module" },
    @{ Path = "docs\modules\lighting.md";       Title = "Lighting & Smart Home Behaviour Module" },
    @{ Path = "docs\modules\media-cinema.md";   Title = "Media / Cinema Integration Module" },
    @{ Path = "docs\modules\security-cctv.md";  Title = "Security, Alarms & CCTV Module" }
)

foreach ($m in $modules) {
    $path  = $m.Path
    $title = $m.Title
    if (-not (Test-Path $path)) {
        Write-Host "Creating module placeholder: $path"
        @(
            "# $title",
            "",
            "## Scope",
            "- TODO: Define what this module covers and where it stops.",
            "",
            "## Responsibilities",
            "- TODO: List responsibilities for this module.",
            "",
            "## Interfaces",
            "- TODO: Describe inputs/outputs, protocols, and integration points.",
            "",
            "## Implementation notes",
            "- TODO: Implementation details and patterns for Gray Logic deployments."
        ) | Set-Content $path -Encoding UTF8
    } else {
        Write-Host "Module already exists: $path (leaving as-is)"
    }
}

# 9. Optional: .editorconfig for consistent formatting
$editorConfig = ".editorconfig"
if (-not (Test-Path $editorConfig)) {
    Write-Host "Creating .editorconfig"
    @(
        "root = true",
        "",
        "[*]",
        "charset = utf-8",
        "end_of_line = lf",
        "insert_final_newline = true",
        "indent_style = space",
        "indent_size = 2",
        "",
        "[*.ps1]",
        "indent_size = 4",
        "",
        "[*.yml]",
        "indent_size = 2",
        "",
        "[*.md]",
        "trim_trailing_whitespace = true"
    ) | Set-Content $editorConfig -Encoding UTF8
}

Write-Host "`n=== Done. Review git status and adjust any placeholders as needed. ==="
