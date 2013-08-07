/*
Copyright 2013 Manu Goyal

Licensed under the Apache License, Version 2.0 (the "License"); you may not use
this file except in compliance with the License.  You may obtain a copy of the
License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed
under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied.  See the License for the
specific language governing permissions and limitations under the License.
 */

requirejs.config({
  paths: {
    'jquery':        'libs/jquery',
    'underscore':    'libs/underscore',
    'backbone':      'libs/backbone',
    'text':          'libs/text/text'
  },

  shim: {
    'jquery': { exports: '$' },
    'underscore': { exports: '_' },

    'backbone': {
      deps: ['jquery', 'underscore'],
      exports: 'Backbone'
    }
  }
});

requirejs(['app'], function(App) {
  App.initialize();
});
