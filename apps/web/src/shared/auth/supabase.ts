import { createClient } from '@supabase/supabase-js'
import type { SupabaseClient } from '@supabase/supabase-js'

import { requiredEnv } from '../config/env'

// TODO: Replace this factory with a lazy shared Supabase client before expanding auth flows.
export function createSupabaseClient(): SupabaseClient {
  return createClient(requiredEnv('VITE_SUPABASE_URL'), requiredEnv('VITE_SUPABASE_ANON_KEY'), {
    auth: {
      flowType: 'pkce'
    }
  })
}
