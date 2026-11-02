package httproot

import "embed"

const (
	// VueUI directory contains common/shared files to create a Vue-SPA
	VueUI = "vue-ui"
)

//go:embed vue-ui/*
var FS embed.FS
