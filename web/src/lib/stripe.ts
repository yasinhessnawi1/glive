import Stripe from 'stripe'

// Lazy initialize Stripe server-side to avoid build-time errors
let stripeInstance: Stripe | null = null

export function getStripeServer(): Stripe | null {
  if (typeof window !== 'undefined') return null

  if (!stripeInstance && process.env.STRIPE_SECRET_KEY) {
    stripeInstance = new Stripe(process.env.STRIPE_SECRET_KEY, {
      apiVersion: '2025-08-27.basil',
      typescript: true,
    })
  }

  return stripeInstance
}

// Export for backwards compatibility - lazy getter
export const stripe = new Proxy({} as Stripe, {
  get(_, prop) {
    const instance = getStripeServer()
    if (!instance) {
      throw new Error('Stripe not initialized - STRIPE_SECRET_KEY may be missing')
    }
    return (instance as any)[prop]
  }
})

// For client-side usage
export const getStripe = () => {
  if (typeof window === 'undefined') return null

  return import('@stripe/stripe-js').then(({ loadStripe }) =>
    loadStripe(process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY!)
  )
}
