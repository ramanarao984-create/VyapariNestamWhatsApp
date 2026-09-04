<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { CalendarDays, Clock3, Info, MessageSquare, StickyNote, X } from 'lucide-vue-next'
import ContactInfoPanel from '@/components/chat/ContactInfoPanel.vue'
import ConversationNotes from '@/components/chat/ConversationNotes.vue'
import { getAvatarGradient, getInitials } from '@/lib/utils'
import type { Contact } from '@/stores/contacts'

interface SessionData {
  session_id?: string
  flow_id?: string
  flow_name?: string
  session_data: Record<string, any>
  panel_config: { sections?: unknown[] }
}

const props = defineProps<{
  contact: Contact
  sessionData?: SessionData | null
  initialTab?: 'summary' | 'notes'
}>()

const emit = defineEmits<{
  close: []
  tagsUpdated: [tags: string[]]
}>()

const activeTab = ref<'summary' | 'notes'>(props.initialTab || 'summary')

watch(() => props.initialTab, (tab) => {
  if (tab) activeTab.value = tab
})

const lastInteraction = computed(() => props.contact.last_message_at || props.contact.last_inbound_at)

const formattedLastInteraction = computed(() => {
  if (!lastInteraction.value) return 'No WhatsApp activity yet'
  const value = new Date(lastInteraction.value)
  if (Number.isNaN(value.getTime())) return 'Recent WhatsApp activity'
  return value.toLocaleString(undefined, { day: 'numeric', month: 'short', hour: 'numeric', minute: '2-digit' })
})

const serviceWindowLabel = computed(() => {
  if (props.contact.service_window_open === true) return 'Reply window open'
  if (props.contact.service_window_open === false) return 'Template may be needed'
  return 'WhatsApp status unavailable'
})
</script>

<template>
  <aside
    aria-label="Patient summary"
    class="w-[420px] max-w-[calc(100vw-3rem)] border-l border-white/[0.08] light:border-gray-200 bg-[#111113] light:bg-white flex flex-col shadow-2xl"
  >
    <div class="px-4 py-3 border-b border-white/[0.08] light:border-gray-200">
      <div class="flex items-start gap-3">
        <Avatar class="h-10 w-10 shrink-0">
          <AvatarImage :src="contact.avatar_url" />
          <AvatarFallback :class="'text-sm bg-gradient-to-br text-white ' + getAvatarGradient(contact.name || contact.phone_number)">
            {{ getInitials(contact.name || contact.phone_number) }}
          </AvatarFallback>
        </Avatar>
        <div class="min-w-0 flex-1">
          <p class="truncate text-sm font-semibold text-white light:text-gray-900">{{ contact.name || contact.phone_number }}</p>
          <p class="mt-0.5 text-xs text-white/45 light:text-gray-500">{{ contact.phone_number }}</p>
        </div>
        <Button
          variant="ghost"
          size="icon"
          class="h-8 w-8 shrink-0 text-white/45 hover:text-white hover:bg-white/[0.08] light:text-gray-500 light:hover:text-gray-900 light:hover:bg-gray-100"
          aria-label="Close patient summary"
          @click="emit('close')"
        >
          <X class="h-4 w-4" />
        </Button>
      </div>
    </div>

    <Tabs v-model="activeTab" class="flex min-h-0 flex-1 flex-col">
      <div class="px-4 pt-3">
        <TabsList class="grid h-9 w-full grid-cols-2 bg-white/[0.06] light:bg-gray-100">
          <TabsTrigger value="summary" class="gap-1.5 text-xs">
            <Info class="h-3.5 w-3.5" /> Summary
          </TabsTrigger>
          <TabsTrigger value="notes" class="gap-1.5 text-xs">
            <StickyNote class="h-3.5 w-3.5" /> Reception notes
          </TabsTrigger>
        </TabsList>
      </div>

      <TabsContent value="summary" class="mt-0 min-h-0 flex-1 overflow-hidden">
        <div class="grid grid-cols-2 gap-2 px-4 py-3">
          <div class="rounded-xl border border-white/[0.08] bg-white/[0.04] p-3 light:border-gray-200 light:bg-gray-50">
            <div class="flex items-center gap-1.5 text-[11px] text-white/45 light:text-gray-500">
              <MessageSquare class="h-3.5 w-3.5" /> WhatsApp
            </div>
            <p class="mt-1 text-xs font-medium text-white/85 light:text-gray-800">{{ serviceWindowLabel }}</p>
          </div>
          <div class="rounded-xl border border-white/[0.08] bg-white/[0.04] p-3 light:border-gray-200 light:bg-gray-50">
            <div class="flex items-center gap-1.5 text-[11px] text-white/45 light:text-gray-500">
              <Clock3 class="h-3.5 w-3.5" /> Last interaction
            </div>
            <p class="mt-1 text-xs font-medium text-white/85 light:text-gray-800">{{ formattedLastInteraction }}</p>
          </div>
        </div>

        <div v-if="contact.tags?.length" class="px-4 pb-2">
          <p class="mb-1.5 text-[11px] font-medium uppercase tracking-wide text-white/35 light:text-gray-500">Patient tags</p>
          <div class="flex flex-wrap gap-1.5">
            <Badge v-for="tag in contact.tags" :key="tag" variant="secondary" class="text-[11px]">{{ tag }}</Badge>
          </div>
        </div>

        <div class="mx-4 mb-2 flex items-center gap-2 rounded-lg border border-emerald-500/15 bg-emerald-500/5 px-3 py-2 text-xs text-emerald-200 light:border-emerald-200 light:bg-emerald-50 light:text-emerald-800">
          <CalendarDays class="h-4 w-4 shrink-0" />
          Appointment history will appear here when the appointment desk is added in the next calendar phase.
        </div>

        <ContactInfoPanel
          :contact="contact"
          :session-data="sessionData"
          embedded
          @tags-updated="emit('tagsUpdated', $event)"
        />
      </TabsContent>

      <TabsContent value="notes" class="mt-0 min-h-0 flex-1 overflow-hidden">
        <ConversationNotes :contact-id="contact.id" embedded @close="emit('close')" />
      </TabsContent>
    </Tabs>
  </aside>
</template>
