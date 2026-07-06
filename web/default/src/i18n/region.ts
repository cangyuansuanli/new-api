/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

export type DeploymentRegion = 'cn' | 'eu'

const EU_HOST_PREFIXES = ['eu-ai.']

/** True when this build or hostname targets the European deployment. */
export function isEuDeployment(): boolean {
  if (import.meta.env.VITE_DEPLOYMENT_REGION === 'eu') {
    return true
  }
  if (typeof window !== 'undefined') {
    const host = window.location.hostname.toLowerCase()
    return EU_HOST_PREFIXES.some((prefix) => host.startsWith(prefix))
  }
  return false
}

/** Separate localStorage key so main-site language prefs do not leak to EU. */
export function getI18nLocalStorageKey(): string {
  return isEuDeployment() ? 'i18nextLng-eu' : 'i18nextLng'
}

/** Initial language for EU; undefined lets the detector run on the main site. */
export function getDeploymentDefaultLanguage(): string | undefined {
  return isEuDeployment() ? 'en' : undefined
}
