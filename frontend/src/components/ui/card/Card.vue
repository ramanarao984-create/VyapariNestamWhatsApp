<script setup lang="ts">
import { cn } from '@/lib/utils'

const props = defineProps<{
  class?: string
}>()
</script>

<template>
  <div
    :class="cn(
      'rounded-2xl border border-white/[0.07] bg-white/[0.035] text-card-foreground backdrop-blur-sm shadow-[0_14px_38px_-28px_rgba(0,0,0,.95)] transition-all duration-300 hover:border-emerald-300/20 hover:bg-white/[0.05] hover:shadow-[0_18px_44px_-28px_rgba(16,185,129,.22)] light:bg-white light:border-slate-200 light:shadow-[0_14px_34px_-28px_rgba(15,23,42,.28)] light:hover:border-emerald-200 light:hover:shadow-[0_18px_42px_-28px_rgba(13,148,136,.24)]',
      props.class
    )"
  >
    <slot />
  </div>
</template>
