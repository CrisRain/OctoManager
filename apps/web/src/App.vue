<template>
  <!-- Skip to main content (a11y: keyboard users bypass nav) -->
  <a href="#main-content" class="skip-link">跳至主要内容</a>

  <div id="main-content" role="main" class="h-full">
    <router-view />
  </div>

  <!-- Live region for screen-reader announcements (toast/status changes) -->
  <div aria-live="polite" aria-atomic="true" class="sr-only" id="aria-announcer" />
</template>
