import JSZip from 'jszip';

export const parseZipTestCases = async (file) => {
  try {
    const zip = new JSZip();
    const loadedZip = await zip.loadAsync(file);
    const inputs = {};
    const outputs = {};
    
    for (const [relativePath, zipEntry] of Object.entries(loadedZip.files)) {
      if (zipEntry.dir || relativePath.includes('__MACOSX') || relativePath.includes('.DS_Store')) continue;
      
      const contentStr = await zipEntry.async("string"); 
      const cleanName = relativePath.split('/').pop().toLowerCase();
      
      const baseMatch = cleanName.match(/(\d+)/); 
      const baseName = baseMatch ? baseMatch[0] : cleanName.split('.')[0];
      
      if (cleanName.includes('in')) inputs[baseName] = contentStr;
      if (cleanName.includes('out')) outputs[baseName] = contentStr;
    }

    const newTestCases = [];
    Object.keys(inputs).forEach(key => {
      if (outputs[key]) {
        const inText = inputs[key].trim();
        const outText = outputs[key].trim();
        
        if (inText.length > 0 || outText.length > 0) {
          newTestCases.push({ input: inText, expectedOutput: outText, isHidden: true });
        }
      }
    });

    if (newTestCases.length === 0) {
      return { error: 'No valid (non-empty) input/output files found in ZIP.' };
    }

    return { testCases: newTestCases };
  } catch (err) {
    return { error: 'Failed to process ZIP file.' };
  }
};