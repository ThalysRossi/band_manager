export type EnvKey = 'VITE_API_BASE_URL' | 'VITE_SUPABASE_URL' | 'VITE_SUPABASE_PUBLISHABLE_KEY'

export function requiredEnv(key: EnvKey): string {
  const value = import.meta.env[key]
  if (value === undefined || value.trim() === '') {
    throw new Error(`${key} is required`)
  }

  return value
}
