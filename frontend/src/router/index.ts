import { createRouter, createWebHistory } from 'vue-router'
import AppLayout from '@/layouts/AppLayout.vue'
import DashboardView from '@/views/DashboardView.vue'
import DocumentDetailView from '@/views/DocumentDetailView.vue'
import DocumentsView from '@/views/DocumentsView.vue'
import KnowledgeBaseListView from '@/views/KnowledgeBaseListView.vue'
import KnowledgeBaseDetailView from '@/views/KnowledgeBaseDetailView.vue'
import KnowledgeAskView from '@/views/KnowledgeAskView.vue'
import QAGenerateView from '@/views/QAGenerateView.vue'
import QAManageView from '@/views/QAManageView.vue'
import QASettingsView from '@/views/QASettingsView.vue'
import SettingsView from '@/views/SettingsView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      component: AppLayout,
      children: [
        { path: '', redirect: '/dashboard' },
        { path: 'dashboard', component: DashboardView },
        { path: 'knowledge-bases', component: KnowledgeBaseListView },
        { path: 'knowledge-bases/:id', component: KnowledgeBaseDetailView },
        { path: 'knowledge-ask', component: KnowledgeAskView },
        { path: 'documents', component: DocumentsView },
        { path: 'documents/:id', component: DocumentDetailView },
        { path: 'qa-generate', component: QAGenerateView },
        { path: 'qa', component: QAManageView },
        { path: 'qa/settings', component: QASettingsView },
        { path: 'settings', component: SettingsView },
      ],
    },
  ],
})

export default router
