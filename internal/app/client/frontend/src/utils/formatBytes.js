export const formatBytes = (bytes) => {
    if (bytes < 0) return `0.00 Kb/s`;
    const kb = bytes / 1024;
    if (kb < 1024) return `${kb.toFixed(2)} Kb/s`;
    const mb = kb / 1024;
    if (mb < 1024) return `${mb.toFixed(2)} Mb/s`;
    const gb = mb / 1024;
    return `${gb.toFixed(2)} Gb/s`;
};