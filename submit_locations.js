const locations = [
	[25.008783098875526, 121.46156128249496],
	[25.01198190198876, 121.45911510787008],
	[25.102041737659224, 121.5503976766184],
	[25.06925687007738, 121.46275972460738],
	[25.044753808381667, 121.52116524938289],
	[25.017534922683456, 121.53965523646437],
	[25.102414513135802, 121.5485378717113],
	[24.981455592017102, 121.45914503942744]
];

const baseURL = 'http://23.146.248.105:8080/api/v1';

async function submitLocation(location) {
	const [lat, lng] = location.map(coord => Number(coord.toFixed(6)));

	try {
		const response = await fetch(`${baseURL}/homeDown`, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				'Authorization': 'Bearer'
			},
			body: JSON.stringify({
				where: [lat, lng],
				type: '網路',
				name: undefined
			})
		});

		if (!response.ok) {
			throw new Error(`HTTP error! status: ${response.status}`);
		}

		console.log(`Successfully submitted location: [${lat}, ${lng}]`);
	} catch (error) {
		console.error(`Error submitting location [${lat}, ${lng}]:`, error);
	}
}

// Submit all locations
async function submitAllLocations() {
	console.log('Starting to submit locations...');
	for (const location of locations) {
		await submitLocation(location);
		// Add a small delay between requests to avoid rate limiting
		await new Promise(resolve => setTimeout(resolve, 1000));
	}
	console.log('Finished submitting all locations');
}

// Run the script
submitAllLocations(); 