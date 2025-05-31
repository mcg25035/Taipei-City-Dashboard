<script setup>
import { useDialogStore } from "../../store/dialogStore";
import DialogContainer from "./DialogContainer.vue";

const dialogStore = useDialogStore();

function handleClose() {
	dialogStore.hideAllDialogs();
	dialogStore.showDialog("whereDown");
}
</script>

<template>
	<DialogContainer dialog="youHomeDown" @on-close="handleClose">
		<div class="youhomedown">
			<h2>你家掛了嗎</h2>
			<div class="youhomedown-control">
				<button
					class="youhomedown-control-cancel"
					@click="dialogStore.hideAllDialogs()"
				>
					沒有
				</button>
				<button
					class="youhomedown-control-confirm"
					@click="handleClose"
				>
					掛了
				</button>
			</div>
		</div>
	</DialogContainer>
</template>

<style scoped lang="scss">
.youhomedown {
	width: 300px;
	text-align: center;

	&-control {
		display: flex;
		justify-content: center;
		gap: 1rem;
		margin-top: 1.5rem;

		&-cancel {
			padding: 6px 18px;
			border-radius: 5px;
			background-color: var(--color-border);
			transition: opacity 0.2s;
			font-size: 1.1rem;
			&:hover {
				opacity: 0.8;
			}
		}

		&-confirm {
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
