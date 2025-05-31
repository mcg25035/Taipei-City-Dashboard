<script setup>
import { useDialogStore } from "../../store/dialogStore";
import DialogContainer from "./DialogContainer.vue";

const dialogStore = useDialogStore();

function handleClose() {
	dialogStore.hideAllDialogs();
}
</script>

<template>
	<DialogContainer dialog="reportSuccess" @on-close="handleClose">
		<div class="reportsuccess">
			<h2>後端選轉<br />資料車更新</h2>
			<div class="reportsuccess-control">
				<button
					class="reportsuccess-control-close"
					@click="handleClose"
				>
					關閉
				</button>
			</div>
		</div>
	</DialogContainer>
</template>

<style scoped lang="scss">
.reportsuccess {
	width: 300px;
	text-align: center;

	&-control {
		display: flex;
		justify-content: center;
		margin-top: 1.5rem;

		&-close {
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
