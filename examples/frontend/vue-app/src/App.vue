<template>
  <n-config-provider :theme="darkTheme">
    <n-message-provider>
      <n-layout has-sider position="absolute" class="app-layout">
        <AppSidebar />
        <n-layout-content class="app-content">
          <router-view />
        </n-layout-content>
      </n-layout>
    </n-message-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { darkTheme } from 'naive-ui'
import { NConfigProvider, NMessageProvider, NLayout, NLayoutContent } from 'naive-ui'
import AppSidebar from './components/layout/AppSidebar.vue'
</script>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
html, body, #app { height: 100%; overflow: hidden; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }
.app-layout { top: 0; left: 0; right: 0; bottom: 0; }
.app-content { overflow: hidden; }
</style>
