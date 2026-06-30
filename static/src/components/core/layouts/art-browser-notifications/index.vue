<script setup lang="ts">
  import { useGalaxyRegistryTimeoutNotification } from '@/hooks/galaxy-registry/useGalaxyRegistryTimeoutNotification'

  defineOptions({ name: 'ArtBrowserNotifications' })

  useGalaxyRegistryTimeoutNotification()
</script>
