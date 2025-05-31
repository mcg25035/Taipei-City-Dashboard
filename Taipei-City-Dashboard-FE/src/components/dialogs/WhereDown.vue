<script setup>
import { ref } from "vue";
import { useDialogStore } from "../../store/dialogStore";
import DialogContainer from "./DialogContainer.vue";

const dialogStore = useDialogStore();
const location = ref("");
const name = ref("");
const longitude = ref("");
const latitude = ref("");

function handleNext() {
	dialogStore.setHomeDownForm({
		longitude: longitude.value,
		latitude: latitude.value,
		name: name.value,
	});
	dialogStore.hideAllDialogs();
	dialogStore.showDialog("whatDown");
}
function handleClose() {
	dialogStore.hideAllDialogs();
}
</script>

<template>
	<DialogContainer dialog="whereDown" @on-close="handleClose">
		<div class="wheredown">
			<h2>哪裡掛的？</h2>
			<input
				v-model="location"
				placeholder="輸入鄰里"
				class="wheredown-input"
			/>
			<input
				v-model="name"
				placeholder="輸入你的代稱"
				class="wheredown-input"
				style="margin-top: 0.5rem"
			/>
			<input
				v-model="longitude"
				placeholder="經度(可選)"
				class="wheredown-input"
				style="margin-top: 0.5rem"
			/>
			<input
				v-model="latitude"
				placeholder="緯度(可選)"
				class="wheredown-input"
				style="margin-top: 0.5rem"
			/>
			<div class="wheredown-control">
				<button class="wheredown-control-next" @click="handleNext">
					下一步
				</button>
			</div>
		</div>
	</DialogContainer>
</template>

<style scoped lang="scss">
.wheredown {
	width: 300px;
	text-align: center;

	&-input {
		width: 80%;
		margin: 1rem 0;
		padding: 6px 10px;
		border-radius: 5px;
		border: 1px solid var(--color-border);
		font-size: 1rem;
	}

	&-control {
		display: flex;
		justify-content: center;
		margin-top: 1rem;

		&-next {
			padding: 6px 18px;
			border-radius: 5px;
			background-color: var(--color-highlight);
			transition: opacity 0.2s;
			font-size: 1.1rem;
			&:hover {
				opacity: 0.8;
			}
		}
	}
}
</style>
