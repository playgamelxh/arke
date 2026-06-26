export function sanitizeParsedText(text: string) {
  let result = text.replace(/\u00a0/g, ' ').replace(/\u200b/g, '').replace(/\ufeff/g, '')
  result = result.replace(/<\/?(p|div|span|br|li|ul|ol|table|tr|td|th|h[1-6])[^>]*>/gi, '\n')
  result = result.replace(/<[^>]+>/g, '')
  result = Array.from(result)
    .filter((ch) => {
      const code = ch.charCodeAt(0)
      return !((code >= 0 && code <= 8) || code === 11 || code === 12 || (code >= 14 && code <= 31) || code === 127)
    })
    .join('')
  result = result.replace(/[\t\r]+/g, ' ')
  result = result.replace(/[•●◦▪■□▶►▸▹◆◇○●★☆]+/g, ' ')
  result = result.replace(/\n{3,}/g, '\n\n')
  result = result.replace(/[ ]{2,}/g, ' ')
  result = result.replace(/^[ \t]+/gm, '').replace(/[ \t]+$/gm, '')
  return result.trim()
}
