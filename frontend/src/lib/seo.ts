export const SITE_NAME = 'SIMPEG'

export const SITE_URL = (
  import.meta.env.VITE_APP_URL ??
  (import.meta.env.DEV ? 'http://localhost:3000' : 'https://simpeg.unisubang.ac.id')
).replace(/\/+$/, '')

export const OG_IMAGE_PATH = '/og-image.png'

export const OG_IMAGE_URL = `${SITE_URL}${OG_IMAGE_PATH}`

export function socialMeta(input: { title: string; description: string; path: string }) {
  const url = `${SITE_URL}${input.path}`
  const image = `${SITE_URL}${OG_IMAGE_PATH}`

  return [
    { property: 'og:type', content: 'website' },
    { property: 'og:site_name', content: SITE_NAME },
    { property: 'og:locale', content: 'id_ID' },
    { property: 'og:title', content: input.title },
    { property: 'og:description', content: input.description },
    { property: 'og:image', content: image },
    { property: 'og:image:alt', content: input.title },
    { property: 'og:url', content: url },
    { name: 'twitter:card', content: 'summary_large_image' },
    { name: 'twitter:title', content: input.title },
    { name: 'twitter:description', content: input.description },
    { name: 'twitter:image', content: image },
    { name: 'twitter:url', content: url },
  ]
}
