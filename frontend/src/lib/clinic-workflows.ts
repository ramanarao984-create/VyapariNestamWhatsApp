export type ClinicMessageLanguage = 'en' | 'te' | 'tenglish'

export interface ClinicTemplatePreset {
  id: string
  name: string
  description: string
  category: 'UTILITY' | 'MARKETING'
  body: Record<ClinicMessageLanguage, string>
  sampleValues: string[]
}

export const clinicTemplatePresets: ClinicTemplatePreset[] = [
  {
    id: 'appointment_confirmation',
    name: 'Appointment confirmation',
    description: 'Confirm the doctor, date, and time after a booking.',
    category: 'UTILITY',
    body: {
      en: 'Hello {{1}}, your appointment with {{2}} is confirmed for {{3}} at {{4}}. Please reply here if you need help.',
      te: 'నమస్కారం {{1}}, {{2}} గారితో మీ అపాయింట్‌మెంట్ {{3}} న {{4}} కి నిర్ధారించబడింది. సహాయం కావాలంటే ఇక్కడే ప్రత్యుత్తరం ఇవ్వండి.',
      tenglish: 'Namaskaram {{1}}, {{2}} gaaritho mee appointment {{3}} na {{4}} ki confirm ayyindi. Help kaavalante ikkade reply ivvandi.',
    },
    sampleValues: ['Priya', 'Dr. Meera', 'Friday, 22 August', '11:00 AM'],
  },
  {
    id: 'appointment_reminder',
    name: 'Appointment reminder',
    description: 'A calm reminder for an upcoming visit.',
    category: 'UTILITY',
    body: {
      en: 'Hello {{1}}, this is a reminder for your appointment with {{2}} tomorrow at {{3}}. Please reply if you need to reschedule.',
      te: 'నమస్కారం {{1}}, {{2}} గారితో మీ అపాయింట్‌మెంట్ రేపు {{3}} కి ఉంది. సమయం మార్చుకోవాలంటే ప్రత్యుత్తరం ఇవ్వండి.',
      tenglish: 'Namaskaram {{1}}, {{2}} gaaritho mee appointment repu {{3}} ki undi. Time marchukovaalante reply ivvandi.',
    },
    sampleValues: ['Priya', 'Dr. Meera', '11:00 AM'],
  },
  {
    id: 'reschedule_follow_up',
    name: 'Reschedule follow-up',
    description: 'Help a patient choose another suitable time.',
    category: 'UTILITY',
    body: {
      en: 'Hello {{1}}, we can help you find another appointment time with {{2}}. Please reply with your preferred day or call {{3}}.',
      te: 'నమస్కారం {{1}}, {{2}} గారితో మరొక అపాయింట్‌మెంట్ సమయం కనుగొనడంలో మేము సహాయం చేస్తాము. మీకు అనుకూలమైన రోజు చెప్పండి లేదా {{3}} కి కాల్ చేయండి.',
      tenglish: 'Namaskaram {{1}}, {{2}} gaaritho vere appointment time kanukkovadaniki memu help chestham. Meeku anukoolamaina roju cheppandi leda {{3}} ki call cheyandi.',
    },
    sampleValues: ['Priya', 'Dr. Meera', '+91 98765 43210'],
  },
  {
    id: 'missed_appointment',
    name: 'Missed appointment follow-up',
    description: 'Invite a patient to rebook after a missed visit.',
    category: 'UTILITY',
    body: {
      en: 'Hello {{1}}, we missed you today. If you would like another appointment with {{2}}, reply here and we will help you rebook.',
      te: 'నమస్కారం {{1}}, ఈరోజు మిమ్మల్ని మిస్ అయ్యాము. {{2}} గారితో మరొక అపాయింట్‌మెంట్ కావాలంటే ఇక్కడే ప్రత్యుత్తరం ఇవ్వండి.',
      tenglish: 'Namaskaram {{1}}, memu mimmalni ivvala miss ayyamu. {{2}} gaaritho vere appointment kaavalante ikkade reply ivvandi.',
    },
    sampleValues: ['Priya', 'Dr. Meera'],
  },
  {
    id: 'feedback_request',
    name: 'Visit feedback request',
    description: 'Ask for a short post-visit experience rating.',
    category: 'MARKETING',
    body: {
      en: 'Hello {{1}}, thank you for visiting {{2}}. We would value your feedback on today’s experience.',
      te: 'నమస్కారం {{1}}, {{2}} ను సందర్శించినందుకు ధన్యవాదాలు. ఈరోజు మీ అనుభవంపై మీ అభిప్రాయం మాకు విలువైనది.',
      tenglish: 'Namaskaram {{1}}, {{2}} ni visit chesinanduku dhanyavaadalu. Ivala mee experience pai mee feedback maaku chaala viluvainadi.',
    },
    sampleValues: ['Priya', 'Greenleaf Clinic'],
  },
]

export const clinicCampaignRecipes = [
  { id: 'appointment_reminder', name: 'Tomorrow’s appointment reminder', description: 'Remind patients about tomorrow’s scheduled visit.', templatePresetId: 'appointment_reminder' },
  { id: 'missed_appointment', name: 'Missed appointment follow-up', description: 'Invite missed patients to rebook.', templatePresetId: 'missed_appointment' },
  { id: 'feedback_request', name: 'Visit feedback request', description: 'Ask recent visitors for feedback.', templatePresetId: 'feedback_request' },
  { id: 'annual_recall', name: 'Annual check-up recall', description: 'Bring eligible patients back for a routine review.', templatePresetId: 'appointment_reminder' },
] as const

export function resolveClinicLanguage(language: ClinicMessageLanguage) {
  return language === 'tenglish' ? 'en' : language
}

export function getClinicTemplatePreset(id?: string) {
  return clinicTemplatePresets.find((preset) => preset.id === id)
}
