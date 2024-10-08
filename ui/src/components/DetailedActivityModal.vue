<template>
  <div>
    <!-- Modal -->
    <div
      id="mapModal"
      class="modal fade modal-dialog-scrollable modal-xl"
      tabindex="-1"
      aria-labelledby="mapModalLabel"
      aria-hidden="true"
    >
      <div class="modal-dialog">
        <div class="modal-content">
          <div class="modal-header">
            <h5
              id="mapModalLabel"
              class="modal-title"
            >
              {{ title() }}
            </h5>
            <button
              type="button"
              class="btn-close"
              aria-label="Close"
              @click="hideModal"
            />
          </div>
          <div class="modal-body">
            <div
              id="map"
              style="width: 100%; height: 400px"
            />
          </div>
          <div class="modal-footer">
            <button
              type="button"
              class="btn btn-secondary"
              aria-label="Close"
              @click="hideModal"
            >
              Close
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, onMounted, onUnmounted, watch } from "vue";
import L from "leaflet";
import "bootstrap/dist/css/bootstrap.min.css";
import "bootstrap";
import { Modal } from "bootstrap";
import { DetailedActivity } from "@/models/activity.model"; // Ensure correct import
import { formatSpeed, formatTime } from "@/utils/formatters";

export default defineComponent({
  name: "DetailedActivityModal",
  props: {
    activity: {
      type: Object as () => DetailedActivity | null,
      required: false,
      default: () => null,
    },
  },

  setup(props) {
    let map: L.Map | null = null;
    let modalInstance: Modal | null = null;    


    let title = () => {
      return props.activity?.name + " - " 
      + (props.activity?.averageSpeed !== undefined ? formatSpeed(props.activity.averageSpeed, props.activity.type) : "N/A")
      + (props.activity?.distance !== undefined ? " / " + props.activity.distance / 1000 + " km" : "")
      + (props.activity?.elapsedTime !== undefined ? " / " + formatTime(props.activity.elapsedTime) : "")
      + (props.activity?.totalElevationGain !== undefined ? " / " + props.activity.totalElevationGain + " m" : "");
    };

    const showModal = () => {
      const modalElement = document.getElementById("mapModal")!;
      modalInstance = new Modal(modalElement);
      modalInstance.show();

      modalElement.addEventListener("shown.bs.modal", () => {
        if (!map) {
          map = L.map("map").setView([51.505, -0.09], 13);
          L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
            attribution:
              '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors',
          }).addTo(map);

          // Add a polyline
          const latlngs = props.activity?.stream?.latitudeLongitude?.map((latlng: number[]) => 
            L.latLng(latlng[0], latlng[1])
          );

          if (latlngs) {
            const polyline = L.polyline(latlngs, { color: "red" }).addTo(map);
            // Fit the map to the bounds of all polylines
            const bounds = L.latLngBounds((polyline.getLatLngs() as L.LatLng[]));
            map.fitBounds(bounds);
          }

        }
      });

      modalElement.addEventListener("hidden.bs.modal", () => {
        if (map) {
          map.remove();
          map = null;
        }
        if (modalInstance) {
          modalInstance.dispose();
          modalInstance = null;
        }
        // Remove backdrop manually if needed
        const backdrop = document.querySelector(".modal-backdrop");
        if (backdrop) {
          backdrop.remove();
        }
      });
    };

    const hideModal = () => {
      const modalElement = document.getElementById("mapModal")!;
      modalInstance = Modal.getInstance(modalElement);
      if (modalInstance) {
        modalInstance.hide();
      }
    };

    onUnmounted(() => {
      if (map) {
        map.remove();
      }
      if (modalInstance) {
        modalInstance.dispose();
      }
    });

    watch(
      () => props.activity,
      (newActivity) => {
        if (newActivity) {
          title = newActivity.name + " - " + formatSpeed(newActivity.averageSpeed, newActivity.type);
          showModal();
        }
      }
    );

    return {
      showModal,
      hideModal,
      title,
    };
  },
});
</script>

<style scoped>
#map {
  width: 100%;
  height: 100%;
  border-radius: 10px; /* Example of custom styling */
}
</style>
