package main

import (
	"embed"
)

// Embed all template files for self-contained binary

//go:embed web/templates/*.html
//go:embed web/templates/partials/common/*.html
//go:embed web/templates/partials/applications/*.html
//go:embed web/templates/partials/vps/*.html
//go:embed web/templates/partials/wizard/*.html
var HTMLTemplates embed.FS

//go:embed web/static/css/*.css
//go:embed web/static/js/vendor/*.js
//go:embed web/static/js/modules/*.js
//go:embed web/static/js/modules/common/*.js
//go:embed web/static/js/*.js
//go:embed web/static/icons/*.png
//go:embed web/static/icons/*.ico
//go:embed web/static/*.webmanifest
var StaticFiles embed.FS

//go:embed configs/applications/*.yaml
//go:embed internal/templates/applications/*.yaml
//go:embed internal/templates/applications/*.sh
//go:embed charts/xanthus-code-server/Chart.yaml
//go:embed charts/xanthus-code-server/values.yaml
//go:embed charts/xanthus-code-server/templates/*.tpl
//go:embed charts/xanthus-code-server/templates/*.yaml
var AllApplicationFiles embed.FS

//go:embed tests/integration/e2e/fixtures/sample_manifests/*.yaml
//go:embed tests/integration/e2e/fixtures/test_configs/*.json
//go:embed tests/integration/e2e/fixtures/mock_responses/*.json
var TestFixtures embed.FS