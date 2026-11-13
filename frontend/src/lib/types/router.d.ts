import type { QueryClient } from '@tanstack/react-query'

/** Konteks router yang diteruskan ke seluruh rute. */
export type RouterContext = {
  queryClient: QueryClient
}
