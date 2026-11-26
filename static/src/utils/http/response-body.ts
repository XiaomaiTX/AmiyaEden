export function decodeBlobResponse(body: Blob): Blob | Promise<unknown> {
  const mediaType = body.type.split(';', 1)[0].trim().toLowerCase()
  if (mediaType !== 'application/json' && !mediaType.endsWith('+json')) return body
  return body.text().then((text) => JSON.parse(text) as unknown)
}
