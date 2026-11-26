const output = (frontend) => ({
  input: process.env.OPENAPI_BUNDLE,
  output: {
    target: `../${frontend}/src/api/generated/client.ts`,
    client: 'axios-functions',
    headers: true,
    baseUrl: { getBaseUrlFromSpecification: true },
    override: {
      // Orval only enables blob mode when the complete success type is Blob.
      // Business errors can share HTTP 200, so retain the union and read by MIME.
      transformer: (operation) => {
        const response = operation.response
        if (response.types.success.some(({ value, formData }) => value === 'Blob' || formData)) {
          return { ...operation, response: { ...response, isBlob: true } }
        }
        return operation
      },
      mutator: {
        path: `../${frontend}/src/api/contract-transport.ts`,
        name: 'contractRequest',
      },
    },
  },
})

module.exports = { vue: output('static'), react: output('static-react') }
