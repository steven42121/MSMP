import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export const useAuthStore = create(
  persist(
    (set) => ({
      token: null,
      user: null,
      tenant: null,
      setAuth: ({ token, user, tenant }) => set({ token, user, tenant }),
      logout: () => set({ token: null, user: null, tenant: null }),
    }),
    { name: 'msmp-auth' }
  )
);
