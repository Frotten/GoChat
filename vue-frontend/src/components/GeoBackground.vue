<template>
  <div class="geo-bg-decor" aria-hidden="true">
    <div class="geo-block geo-block--a"></div>
    <div class="geo-block geo-block--b"></div>
    <div v-if="extra" class="geo-block geo-block--c"></div>
    <div class="geo-flow-line geo-flow-line--h"></div>
    <div class="geo-flow-line geo-flow-line--v"></div>
    <div class="geo-flow-line geo-flow-line--d"></div>
    <div class="geo-orb geo-orb--1"></div>
    <div class="geo-orb geo-orb--2"></div>
  </div>
</template>

<script>
export default {
  name: 'GeoBackground',
  props: {
    extra: {
      type: Boolean,
      default: false
    }
  }
}
</script>
