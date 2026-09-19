<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { TagBadge } from '@/components/ui/tag-badge'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from '@/components/ui/command'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { contactsService, accountsService, type Tag } from '@/services/api'
import { useTagsStore } from '@/stores/tags'
import { toast } from 'vue-sonner'
import { Loader2, Check, ChevronsUpDown, X } from 'lucide-vue-next'
import { getErrorMessage } from '@/lib/api-utils'
import { getTagColorClass } from '@/lib/constants'

const { t } = useI18n()
const tagsStore = useTagsStore()

interface Props {
  open: boolean
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'update:open': [value: boolean]
  'created': [contact: any]
}>()

interface ContactFormData {
  phone_number: string
  profile_name: string
  whatsapp_account: string
  tags: string[]
  profile: {
    email: string
    date_of_birth: string
    gender: string
    area: string
    occupation: string
    lifecycle_stage: string
    acquisition_source: string
    preferred_payment_mode: string
    preferred_language: string
    preferred_contact_method: string
    preferred_visit_time: string
    preferred_practitioner: string
    marketing_consent: boolean
  }
}

const defaultFormData: ContactFormData = {
  phone_number: '', profile_name: '', whatsapp_account: '', tags: [],
  profile: { email: '', date_of_birth: '', gender: '', area: '', occupation: '', lifecycle_stage: 'new_lead', acquisition_source: 'walk_in', preferred_payment_mode: '', preferred_language: '', preferred_contact_method: '', preferred_visit_time: '', preferred_practitioner: '', marketing_consent: false }
}

const formData = ref<ContactFormData>(structuredClone(defaultFormData))
const formError = ref('')
const isSubmitting = ref(false)
const tagSelectorOpen = ref(false)
const availableTags = ref<Tag[]>([])
const availableAccounts = ref<{ id: string; name: string; phone_number: string }[]>([])
const additionalDetailsOpen = ref(false)

watch(() => props.open, (isOpen) => {
  if (isOpen) {
    formData.value = structuredClone(defaultFormData)
    formError.value = ''
    additionalDetailsOpen.value = false
    fetchTags()
    fetchAccounts()
  }
})

async function fetchTags() {
  try {
    const response = await tagsStore.fetchTags({ limit: 100 })
    availableTags.value = response.tags
  } catch {
    // Silently fail - tags are optional
  }
}

async function fetchAccounts() {
  try {
    const response = await accountsService.list()
    const data = response.data as any
    const responseData = data.data || data
    availableAccounts.value = responseData.accounts || []
  } catch (error) {
    // Silently fail - accounts are optional
  }
}

async function saveContact() {
  if (isSubmitting.value) return
  formError.value = ''
  if (!formData.value.phone_number.trim()) {
    toast.error(t('contacts.phoneRequired'))
    return
  }
  isSubmitting.value = true
  try {
    const response = await contactsService.create({
      phone_number: formData.value.phone_number.trim(),
      profile_name: formData.value.profile_name.trim() || undefined,
      whatsapp_account: formData.value.whatsapp_account || undefined,
      tags: formData.value.tags.length > 0 ? formData.value.tags : undefined,
      profile: formData.value.profile
    })
    const contact = response.data?.data || response.data
    toast.success(t('common.createdSuccess', { resource: t('resources.Contact') }))
    emit('update:open', false)
    emit('created', contact)
  } catch (error) {
    formError.value = getErrorMessage(error, t('common.failedCreate', { resource: t('resources.contact') }))
  } finally {
    isSubmitting.value = false
  }
}

function toggleTag(tagName: string) {
  const index = formData.value.tags.indexOf(tagName)
  if (index === -1) {
    formData.value.tags.push(tagName)
  } else {
    formData.value.tags.splice(index, 1)
  }
}

function removeTag(tagName: string) {
  formData.value.tags = formData.value.tags.filter(t => t !== tagName)
}

function isTagSelected(tagName: string): boolean {
  return formData.value.tags.includes(tagName)
}

function getTagDetails(tagName: string): Tag | undefined {
  return availableTags.value.find(t => t.name === tagName)
}

function closeDialog() {
  emit('update:open', false)
}
</script>

<template>
  <Dialog :open="open" @update:open="emit('update:open', $event)">
    <DialogContent class="sm:max-w-xl">
      <DialogHeader>
        <DialogTitle>{{ $t('contacts.createTitle') }}</DialogTitle>
        <DialogDescription>{{ $t('contacts.createDesc') }}</DialogDescription>
      </DialogHeader>
      <div class="space-y-4 py-4">
        <div class="space-y-2">
          <Label for="field-formData-phone-number">{{ $t('contacts.phoneNumber') }} <span class="text-destructive">*</span></Label>
          <Input id="field-formData-phone-number" type="tel" autocomplete="tel" v-model="formData.phone_number" :placeholder="$t('contacts.phonePlaceholder')" />
          <p class="text-xs text-muted-foreground">{{ $t('contacts.phoneHint') }}</p>
        </div>
        <div class="space-y-2">
          <Label for="field-formData-profile-name">{{ $t('contacts.profileName') }}</Label>
          <Input id="field-formData-profile-name" v-model="formData.profile_name" :placeholder="$t('contacts.namePlaceholder')" />
        </div>
        <div v-if="availableAccounts.length > 0" class="space-y-2">
          <Label>{{ $t('contacts.whatsappAccount') }}</Label>
          <Select v-model="formData.whatsapp_account">
            <SelectTrigger>
              <SelectValue :placeholder="$t('contacts.selectAccount')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem v-for="account in availableAccounts" :key="account.id" :value="account.name">
                {{ account.name }} ({{ account.phone_number }})
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div v-if="availableTags.length > 0" class="space-y-2">
          <Label>{{ $t('contacts.tags') }}</Label>
          <Popover v-model:open="tagSelectorOpen">
            <PopoverTrigger as-child>
              <Button variant="outline" role="combobox" class="w-full justify-between">
                <span v-if="formData.tags.length === 0" class="text-muted-foreground">{{ $t('contacts.selectTags') }}</span>
                <span v-else>{{ formData.tags.length }} {{ $t('contacts.tagsSelected') }}</span>
                <ChevronsUpDown class="ml-2 h-4 w-4 shrink-0 opacity-50" />
              </Button>
            </PopoverTrigger>
            <PopoverContent class="w-[300px] p-0" @interact-outside="(e) => e.preventDefault()">
              <Command>
                <CommandInput :placeholder="$t('contacts.searchTags')" />
                <CommandList>
                  <CommandEmpty>{{ $t('contacts.noTagsFound') }}</CommandEmpty>
                  <CommandGroup>
                    <CommandItem
                      v-for="tag in availableTags"
                      :key="tag.name"
                      :value="tag.name"
                      class="flex items-center gap-2 cursor-pointer"
                      @select.prevent="toggleTag(tag.name)"
                    >
                      <div class="flex items-center gap-2 flex-1">
                        <span :class="['w-2 h-2 rounded-full', getTagColorClass(tag.color).split(' ')[0]]"></span>
                        <span>{{ tag.name }}</span>
                      </div>
                      <Check v-if="isTagSelected(tag.name)" class="h-4 w-4 text-primary" />
                    </CommandItem>
                  </CommandGroup>
                </CommandList>
              </Command>
            </PopoverContent>
          </Popover>
          <div v-if="formData.tags.length > 0" class="flex flex-wrap gap-1 mt-2">
            <TagBadge
              v-for="tagName in formData.tags"
              :key="tagName"
              :color="getTagDetails(tagName)?.color"
            >
              {{ tagName }}
              <button
                type="button"
                class="ml-1 rounded-full hover:bg-black/10 dark:hover:bg-white/10 p-0.5 transition-colors"
                @click.stop="removeTag(tagName)"
              >
                <X class="h-3 w-3" />
              </button>
            </TagBadge>
          </div>
        </div>
        <Collapsible v-model:open="additionalDetailsOpen" class="rounded-xl border border-border/70 bg-muted/20">
          <CollapsibleTrigger class="flex w-full items-center justify-between px-3 py-3 text-left text-sm font-medium">
            <span>{{ $t('contacts.additionalDetails') }}</span>
            <span class="text-xs font-normal text-muted-foreground">{{ $t('contacts.optional') }}</span>
          </CollapsibleTrigger>
          <CollapsibleContent class="border-t border-border/70 px-3 pb-3 pt-3 space-y-3">
            <p class="text-xs text-muted-foreground">{{ $t('contacts.additionalDetailsHint') }}</p>
            <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div class="space-y-2"><Label for="field-formData-profile-email">{{ $t('contacts.email') }}</Label><Input id="field-formData-profile-email" v-model="formData.profile.email" type="email" placeholder="name@example.com" /></div>
              <div class="space-y-2"><Label for="field-formData-profile-date-of-birth">{{ $t('contacts.dateOfBirth') }}</Label><Input id="field-formData-profile-date-of-birth" v-model="formData.profile.date_of_birth" type="date" /></div>
              <div class="space-y-2"><Label>{{ $t('contacts.gender') }}</Label><Select v-model="formData.profile.gender"><SelectTrigger><SelectValue :placeholder="$t('contacts.selectOptional')" /></SelectTrigger><SelectContent><SelectItem value="female">Female</SelectItem><SelectItem value="male">Male</SelectItem><SelectItem value="other">Other</SelectItem><SelectItem value="prefer_not_to_say">Prefer not to say</SelectItem></SelectContent></Select></div>
              <div class="space-y-2"><Label for="field-formData-profile-area">{{ $t('contacts.area') }}</Label><Input id="field-formData-profile-area" v-model="formData.profile.area" :placeholder="$t('contacts.areaPlaceholder')" /></div>
              <div class="space-y-2"><Label>{{ $t('contacts.lifecycleStage') }}</Label><Select v-model="formData.profile.lifecycle_stage"><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="new_lead">New lead</SelectItem><SelectItem value="appointment_booked">Appointment booked</SelectItem><SelectItem value="active_patient">Active patient</SelectItem><SelectItem value="follow_up">Follow-up</SelectItem><SelectItem value="inactive">Inactive</SelectItem></SelectContent></Select></div>
              <div class="space-y-2"><Label>{{ $t('contacts.source') }}</Label><Select v-model="formData.profile.acquisition_source"><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="walk_in">Walk-in</SelectItem><SelectItem value="whatsapp">WhatsApp</SelectItem><SelectItem value="phone_call">Phone call</SelectItem><SelectItem value="email">Email</SelectItem><SelectItem value="referral">Referral</SelectItem><SelectItem value="other">Other</SelectItem></SelectContent></Select></div>
              <div class="space-y-2"><Label>{{ $t('contacts.preferredPaymentMode') }}</Label><Select v-model="formData.profile.preferred_payment_mode"><SelectTrigger><SelectValue :placeholder="$t('contacts.selectOptional')" /></SelectTrigger><SelectContent><SelectItem value="upi">UPI</SelectItem><SelectItem value="card">Card</SelectItem><SelectItem value="cash">Cash</SelectItem><SelectItem value="insurance">Insurance</SelectItem><SelectItem value="other">Other</SelectItem></SelectContent></Select></div>
              <div class="space-y-2"><Label for="field-formData-profile-occupation">{{ $t('contacts.occupation') }}</Label><Input id="field-formData-profile-occupation" v-model="formData.profile.occupation" :placeholder="$t('contacts.occupationPlaceholder')" /></div>
              <div class="space-y-2"><Label>{{ $t('contacts.preferredLanguage') }}</Label><Select v-model="formData.profile.preferred_language"><SelectTrigger><SelectValue :placeholder="$t('contacts.selectOptional')" /></SelectTrigger><SelectContent><SelectItem value="english">English</SelectItem><SelectItem value="telugu">Telugu</SelectItem><SelectItem value="tenglish">Tenglish</SelectItem><SelectItem value="hindi">Hindi</SelectItem><SelectItem value="other">Other</SelectItem></SelectContent></Select></div>
              <div class="space-y-2"><Label>{{ $t('contacts.preferredContactMethod') }}</Label><Select v-model="formData.profile.preferred_contact_method"><SelectTrigger><SelectValue :placeholder="$t('contacts.selectOptional')" /></SelectTrigger><SelectContent><SelectItem value="whatsapp">WhatsApp</SelectItem><SelectItem value="phone_call">Phone call</SelectItem><SelectItem value="either">Either is fine</SelectItem></SelectContent></Select></div>
              <div class="space-y-2"><Label>{{ $t('contacts.preferredVisitTime') }}</Label><Select v-model="formData.profile.preferred_visit_time"><SelectTrigger><SelectValue :placeholder="$t('contacts.selectOptional')" /></SelectTrigger><SelectContent><SelectItem value="morning">Morning</SelectItem><SelectItem value="afternoon">Afternoon</SelectItem><SelectItem value="evening">Evening</SelectItem><SelectItem value="any">Any time</SelectItem></SelectContent></Select></div>
              <div class="space-y-2"><Label for="field-formData-profile-preferred-practitioner">{{ $t('contacts.preferredPractitioner') }}</Label><Input id="field-formData-profile-preferred-practitioner" v-model="formData.profile.preferred_practitioner" :placeholder="$t('contacts.preferredPractitionerPlaceholder')" /></div>
            </div>
            <label class="flex items-start gap-2 rounded-lg bg-background/60 px-2 py-2 text-xs text-muted-foreground"><input v-model="formData.profile.marketing_consent" type="checkbox" class="mt-0.5 h-4 w-4" /><span>{{ $t('contacts.marketingConsent') }}</span></label>
          </CollapsibleContent>
        </Collapsible>
      </div>
      <p v-if="formError" role="alert" class="text-sm text-destructive">{{ formError }}</p>
      <div class="flex justify-end gap-2">
        <Button variant="outline" @click="closeDialog">{{ $t('common.cancel') }}</Button>
        <Button @click="saveContact" :disabled="isSubmitting">
          <Loader2 v-if="isSubmitting" class="h-4 w-4 mr-2 animate-spin" />
          {{ $t('common.create') }}
        </Button>
      </div>
    </DialogContent>
  </Dialog>
</template>
