/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

export type DocsNavItem = {
  id: string
  titleKey: string
}

export const docsNavItems: DocsNavItem[] = [
  { id: 'overview', titleKey: 'userDocs.nav.overview' },
  { id: 'quick-start', titleKey: 'userDocs.nav.quickStart' },
  { id: 'api-key', titleKey: 'userDocs.nav.apiKey' },
  { id: 'wallet', titleKey: 'userDocs.nav.wallet' },
  { id: 'models', titleKey: 'userDocs.nav.models' },
  { id: 'chat-image-video', titleKey: 'userDocs.nav.chatImageVideo' },
  { id: 'integrations', titleKey: 'userDocs.nav.integrations' },
  { id: 'infinite-canvas', titleKey: 'userDocs.nav.infiniteCanvas' },
  { id: 'faq', titleKey: 'userDocs.nav.faq' },
]
