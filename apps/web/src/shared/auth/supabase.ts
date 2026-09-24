import { createClient } from '@supabase/supabase-js'
import type { SupabaseClient } from '@supabase/supabase-js'

import { requiredEnv } from '../config/env'

let supabaseClient: SupabaseClient | null = null

export function getSupabaseClient(): SupabaseClient {
  if (supabaseClient === null) {
    supabaseClient = createClient(
      requiredEnv('VITE_SUPABASE_URL'),
      requiredEnv('VITE_SUPABASE_PUBLISHABLE_KEY'),
      {
        auth: {
          flowType: 'pkce'
        }
      }
    )
  }

  return supabaseClient
}
