<script lang="ts" setup>
import { Modal } from 'bootstrap'
import {onBeforeUnmount, onMounted, ref, toRefs,} from 'vue'

const props = withDefaults(
    // eslint-disable-next-line max-len
    defineProps<{ modalClass?: string, noClose?: boolean, okOnly?: boolean, closeOnly?: boolean, okDisabled?: boolean, modalSize?: string, showFooter?: boolean, scrollable?: boolean, validBtnText?: string, cancelBtnText?: string, closeBtnText?: string }>(),
    {
        modalClass: '',
        noClose: false,
        okOnly: false,
        closeOnly: false,
        okDisabled: false,
        modalSize: '',
        showFooter: true,
        scrollable: false,
        validBtnText: 'UTILS.VALID',
        cancelBtnText: 'UTILS.CANCEL',
        closeBtnText: 'UTILS.CLOSE',
    },
)
const { okOnly, closeOnly, okDisabled } = toRefs(props)
const emit = defineEmits(['ok', 'close'])
const modalEl = ref<HTMLElement>()
let boostedModal: Modal

onMounted(() => {
    if (modalEl.value) {
        const options: Partial<Modal.Options> = {}

        if (props.noClose) {
            options.keyboard = false
            options.backdrop = 'static'
        }

        boostedModal = new Modal(modalEl.value, options)

        // Emit close event on close modal (click outside, close button, ...)
        modalEl.value.addEventListener('hidden.bs.modal', () => {
            emit('close')
        })

        boostedModal.show()
    }
})

onBeforeUnmount(() => {
    if (boostedModal) {
        boostedModal.hide()
    }
})
</script>

<template>
  <teleport to="#app">
    <div
      ref="modalEl"
      :class="['modal', modalClass]"
      tabindex="-1"
    >
      <div
        :class="[(modalSize ? `modal-${modalSize}` : ''), { 'modal-dialog-scrollable': scrollable }, 'modal-dialog']"
      >
        <div class="modal-content">
          <div class="modal-header">
            <slot name="modal-title" />
            <button
              v-if="!noClose"
              type="button"
              class="btn-close"
              data-bs-dismiss="modal"
              aria-label="Close"
              @click="emit('close')"
            />
          </div>
          <div
            ref="body"
            class="modal-body"
          >
            <slot />
          </div>
          <div
            v-if="showFooter"
            class="modal-footer"
          >
            <slot name="modal-footer">
              <template v-if="okOnly">
                <button
                  type="button"
                  class="btn btn-primary"
                  :disabled="okDisabled"
                  @click="emit('ok')"
                >
                  OK
                </button>
              </template>
              <template v-else-if="closeOnly">
                <button
                  type="button"
                  class="btn btn-outline-secondary"
                  data-bs-dismiss="modal"
                >
                  Close
                </button>
              </template>
              <template v-else>
                <button
                  type="button"
                  class="btn btn-outline-secondary"
                  data-bs-dismiss="modal"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  class="btn btn-primary"
                  :disabled="okDisabled"
                  @click="emit('ok')"
                >
                  Validate
                </button>
              </template>
            </slot>
          </div>
        </div>
      </div>
    </div>
  </teleport>
</template>
