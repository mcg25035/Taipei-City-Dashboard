<script setup>
import { ref } from "vue";
import { useDialogStore } from "../../store/dialogStore";
import DialogContainer from "./DialogContainer.vue";
import http from "../../router/axios";

const dialogStore = useDialogStore();

const what = ref("");
const options = ["網路", "水", "電", "瓦斯", "其他"];

async function handleSubmit() {
	const { homeDownForm } = dialogStore;
	try {
		await http.post("/homeDown", {
			where: [121.46585368626272, 25.012875478644947],
			type: what.value,
			name: homeDownForm.name,
		});
		dialogStore.hideAllDialogs();
		dialogStore.showDialog("reportSuccess");
	} catch {
		dialogStore.hideAllDialogs();
	}
}

function handleClose() {
	dialogStore.hideAllDialogs();
}
</script>

<template>
	<DialogContainer dialog="whatDown" @on-close="handleClose">
		<div class="whatdown">
			<h2>什麼掛了？</h2>
			<select v-model="what" class="whatdown-select">
				<option value="" disabled>請選擇</option>
				<option v-for="item in options" :key="item" :value="item">
					{{ item }}
				</option>
			</select>
			<div class="whatdown-control">
				<button class="whatdown-control-submit" @click="handleSubmit">
					送出
				</button>
			</div>
		</div>
	</DialogContainer>
</template>

<style scoped lang="scss">
.whatdown {
	width: 300px;
	text-align: center;

	&-select {
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

		&-submit {
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
