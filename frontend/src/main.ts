import { createApp } from 'vue'
import 'open-props/style'
import 'open-props/animations'
import './styles/tokens.css'
import App from './App.vue'
import { applyTheme } from './composables/useTheme'

applyTheme()

createApp(App).mount('#app')
