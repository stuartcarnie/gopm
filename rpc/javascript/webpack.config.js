module.exports = {
    mode: "production",
    entry: "./gopm.ts",
    output: {
        filename: "../../webgui/js/bundle.js",
        libraryTarget: "var",
        library: "gopm",
        path: __dirname
    },
    devtool: "source-map"
}