<!--
SPDX-FileCopyrightText: The matrix-alertmanager-receiver Authors
SPDX-License-Identifier: GPL-3.0-or-later
 -->

This file contains migration guidelines for updating the matrix-alertmanager-receiver on your systems.

# Starting with version 2025.9.17

- The `templating.computed-values` feature is DEPRECATED and will be removed in a future version. Use Go template [variable assignments](https://pkg.go.dev/text/template#hdr-Variables) instead.
