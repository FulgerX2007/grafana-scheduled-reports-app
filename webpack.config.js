const path = require('path');
const CopyWebpackPlugin = require('copy-webpack-plugin');
const ForkTsCheckerWebpackPlugin = require('fork-ts-checker-webpack-plugin');

module.exports = (env, argv) => {
  const isProduction = argv.mode === 'production';

  return {
    target: 'web',
    context: path.join(__dirname, 'src'),
    entry: {
      module: './module.ts',
    },
    output: {
      filename: '[name].js',
      path: path.join(__dirname, 'dist'),
      libraryTarget: 'amd',
      clean: {
        keep: /gpx_reporting/,
      },
    },
    externals: [
      'lodash',
      'react',
      'react-dom',
      '@grafana/data',
      '@grafana/ui',
      '@grafana/runtime',
      (context, request, callback) => {
        const prefix = 'grafana/';
        if (request.indexOf(prefix) === 0) {
          return callback(null, request.substr(prefix.length));
        }
        callback();
      },
    ],
    plugins: [
      new ForkTsCheckerWebpackPlugin({
        typescript: {
          configFile: path.join(__dirname, 'tsconfig.json'),
        },
      }),
      new CopyWebpackPlugin({
        patterns: [
          { from: 'plugin.json', to: '.' },
          { from: 'img/**/*', to: '.' },
          { from: '../README.md', to: '.' },
          { from: '../LICENSE', to: '.', noErrorOnMissing: true },
          { from: '../CHANGELOG.md', to: '.', noErrorOnMissing: true },
        ],
      }),
    ],
    resolve: {
      extensions: ['.ts', '.tsx', '.js', '.jsx'],
      modules: [path.resolve(__dirname, 'src'), 'node_modules'],
    },
    module: {
      rules: [
        {
          test: /\.(ts|tsx)$/,
          exclude: /node_modules/,
          use: {
            loader: 'swc-loader',
            options: {
              jsc: {
                parser: {
                  syntax: 'typescript',
                  tsx: true,
                  decorators: false,
                  dynamicImport: true,
                },
                target: 'es2015',
                transform: {
                  react: {
                    runtime: 'automatic',
                  },
                },
              },
            },
          },
        },
        {
          test: /\.css$/,
          use: ['style-loader', 'css-loader'],
        },
        {
          test: /\.(png|jpe?g|gif|svg)$/,
          type: 'asset/resource',
          generator: {
            filename: 'img/[name][ext]',
          },
        },
      ],
    },
    optimization: {
      minimize: isProduction,
    },
    devtool: isProduction ? 'source-map' : 'eval-source-map',
  };
};
